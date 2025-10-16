package reports

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func BuildFile(reg *mount.Registry, id, ruta string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("rep file: -id requerido")
	}
	ruta = normalizePath(ruta)
	if ruta == "" || ruta == "/" {
		return nil, fmt.Errorf("rep file: -ruta requerida")
	}

	mp, ok := reg.GetByID(id)
	if !ok {
		return nil, fmt.Errorf("rep file: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return nil, fmt.Errorf("rep file: leyendo super bloque: %w", err)
	}

	_, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return nil, fmt.Errorf("rep file: cargando bitmaps: %w", err)
	}

	inoIdx, err := resolvePathToInodeAdaptive(mp, sb, bmBl, ruta)
	if err != nil {
		return nil, fmt.Errorf("rep file: %v", err)
	}

	ino, err := readInodeAt(mp, sb, inoIdx)
	if err != nil {
		return nil, fmt.Errorf("rep file: leyendo inodo #%d: %w", inoIdx, err)
	}

	data, err := readWholeFile(mp, sb, ino, bmBl)
	if err != nil {
		return nil, fmt.Errorf("rep file: leyendo contenido: %w", err)
	}
	if int64(len(data)) > int64(ino.ISize) && ino.ISize >= 0 {
		data = data[:ino.ISize]
	}

	t := decodeType(ino.IType)
	if t != "file" && len(bytes.TrimRight(data, "\x00")) == 0 {
		return nil, fmt.Errorf("rep file: %s no es un archivo regular (tipo=%s)", ruta, t)
	}

	return data, nil
}

func GenerateFile(reg *mount.Registry, id, ruta, outPath string) error {
	data, err := BuildFile(reg, id, ruta)
	if err != nil {
		return err
	}
	final, err := resolveOutPathFile(outPath, id, ruta)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return fmt.Errorf("rep file: creando carpeta destino: %w", err)
	}
	return os.WriteFile(final, data, 0o644)
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

func resolveOutPathFile(out, id, ruta string) (string, error) {
	out = strings.TrimSpace(out)
	base := strings.TrimSpace(filepath.Base(ruta))
	if base == "" || base == "/" || base == "." {
		base = "file_" + id + ".txt"
	} else {
		if strings.EqualFold(filepath.Ext(base), ".txt") {
		} else if filepath.Ext(base) == "" {
			base = base + ".txt"
		}
	}
	if out == "" {
		return base, nil
	}
	ext := strings.ToLower(filepath.Ext(out))
	if ext == "" {
		if st, err := os.Stat(out); err == nil && st.IsDir() {
			return filepath.Join(out, base), nil
		}
		return out + ".txt", nil
	}
	if st, err := os.Stat(out); err == nil && st.IsDir() {
		return filepath.Join(out, base), nil
	}
	return out, nil
}

func resolvePathToInodeAdaptive(mp *mount.MountedPartition, sb ext2.SuperBloque, bmBl []byte, ruta string) (int32, error) {
	parts := splitPath(ruta)
	if len(parts) == 0 {
		return -1, fmt.Errorf("ruta vacía")
	}

	tryFrom := func(rootIdx int32) (int32, error) {
		curIdx := rootIdx
		for i, name := range parts {
			ino, err := readInodeAt(mp, sb, curIdx)
			if err != nil {
				return -1, fmt.Errorf("leyendo inodo %d: %w", curIdx, err)
			}
			isLast := i == len(parts)-1
			if !isLast && !(decodeType(ino.IType) == "dir" || seemsDir(mp, sb, ino)) {
				return -1, fmt.Errorf("ruta intermedia no es carpeta: %q", name)
			}

			children := readDirChildrenAllPointers(mp, sb, ino, bmBl)

			nextIdx := int32(-1)
			want := strings.TrimSpace(name)
			for _, ch := range children {
				if strings.TrimSpace(ch.Name) == want {
					nextIdx = ch.Inode
					break
				}
			}
			if nextIdx < 0 {
				return -1, fmt.Errorf("no existe: %q", name)
			}
			curIdx = nextIdx
		}
		return curIdx, nil
	}

	if idx, err := tryFrom(2); err == nil {
		return idx, nil
	}
	if idx, err := tryFrom(0); err == nil {
		return idx, nil
	}
	return -1, fmt.Errorf("no se pudo resolver ruta %q desde #2 ni #0", ruta)
}

func splitPath(p string) []string {
	p = normalizePath(p)
	raw := strings.Split(p, "/")
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func readWholeFile(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo, bmBl []byte) ([]byte, error) {
	var buf bytes.Buffer
	readDataBlock := func(b int32) {
		if !isLikelyUsedBlock(mp, sb, b, bmBl) {
			return
		}
		raw, err := readBlockBytes(mp, sb, b)
		if err == nil {
			_, _ = buf.Write(raw)
		}
	}
	for i, b := range ino.IBlock {
		if b < 0 || b >= sb.SBlocksCount {
			continue
		}
		switch {
		case i < 12:
			readDataBlock(b)
		case i == 12:
			for _, db := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				readDataBlock(db)
			}
		case i == 13:
			for _, mid := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				for _, db := range readPtrBlockTolerant(mp, sb, mid, bmBl) {
					readDataBlock(db)
				}
			}
		}
	}
	return buf.Bytes(), nil
}

func seemsDir(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo) bool {
	if len(ino.IBlock) == 0 || ino.IBlock[0] < 0 {
		return false
	}
	raw, err := readBlockBytes(mp, sb, ino.IBlock[0])
	if err != nil {
		return false
	}
	ents := parseDirEntries(raw)
	return len(ents) > 0
}
