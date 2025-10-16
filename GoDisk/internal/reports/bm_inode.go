package reports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func GenerateBmInode(reg *mount.Registry, id, outPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("rep bm_inode: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rep bm_inode: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return fmt.Errorf("rep bm_inode: leyendo super bloque: %w", err)
	}

	n := int(sb.SInodesCount)
	if n <= 0 {
		return fmt.Errorf("rep bm_inode: SInodesCount inválido: %d", n)
	}
	bmIn, err := readBytesAt(mp.DiskPath, mp.Start+sb.SBmInodeStart, n)
	if err != nil {
		return fmt.Errorf("rep bm_inode: leyendo bitmap: %w", err)
	}

	const perLine = 20
	var b strings.Builder
	for i := 0; i < n; i++ {
		if bmIn[i] != 0 {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}

		if (i+1)%perLine == 0 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
	if n%perLine != 0 {
		b.WriteByte('\n')
	}

	finalPath := resolveOutPathBmInode(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep bm_inode: creando carpeta destino: %w", err)
	}
	return os.WriteFile(finalPath, []byte(b.String()), 0o644)
}

// ---- helpers locales ----

func resolveOutPathBmInode(out, id string) string {
	out = strings.TrimSpace(out)
	def := fmt.Sprintf("bm_inode_%s.txt", id)
	if out == "" {
		return def
	}
	ext := strings.ToLower(filepath.Ext(out))
	if ext == "" {
		if st, err := os.Stat(out); err == nil && st.IsDir() {
			return filepath.Join(out, def)
		}
		return out + ".txt"
	}
	return out
}

func BuildBmInodeText(reg *mount.Registry, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("rep bm_inode: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return "", fmt.Errorf("rep bm_inode: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return "", fmt.Errorf("rep bm_inode: leyendo super bloque: %w", err)
	}
	n := int(sb.SInodesCount)
	if n <= 0 {
		return "", fmt.Errorf("rep bm_inode: SInodesCount inválido: %d", n)
	}

	bmIn, err := readBytesAt(mp.DiskPath, mp.Start+sb.SBmInodeStart, n)
	if err != nil {
		return "", fmt.Errorf("rep bm_inode: leyendo bitmap: %w", err)
	}

	const perLine = 20
	var b strings.Builder
	for i := 0; i < n; i++ {
		if bmIn[i] != 0 {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		if (i+1)%perLine == 0 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
	if n%perLine != 0 {
		b.WriteByte('\n')
	}
	return b.String(), nil
}
