package reports

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// ---------- Modelo JSON para Angular ----------

type InodeReport struct {
	Kind       string  `json:"kind"`
	DiskPath   string  `json:"diskPath"`
	ID         string  `json:"id"`
	Index      int32   `json:"index"`
	Type       string  `json:"type"`
	RawType    byte    `json:"rawType"`
	Size       int32   `json:"size"`
	UID        int32   `json:"uid"`
	GID        int32   `json:"gid"`
	Perm       string  `json:"perm"`
	PermRaw    []byte  `json:"permRaw"`
	ATime      string  `json:"atime"`
	MTime      string  `json:"mtime"`
	CTime      string  `json:"ctime"`
	BlocksUsed int     `json:"blocksUsed"`
	Blocks     []int32 `json:"blocks"`
}

func GenerateInode(reg *mount.Registry, id, ruta, outPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("rep inode: -id requerido")
	}

	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rep inode: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return fmt.Errorf("rep inode: leyendo super bloque: %w", err)
	}
	if sb.SInodesCount <= 0 {
		return fmt.Errorf("rep inode: la partición con id %s no está formateada (SInodesCount=0). Ejecute mkfs -id=%s", id, id)
	}

	ruta = strings.TrimSpace(ruta)
	var inIdx int32
	if ruta == "" || ruta == "/" || strings.EqualFold(ruta, "root") {

		inIdx = 0
	} else {
		n, err := strconv.ParseInt(ruta, 10, 32)
		if err != nil {
			return /* DiskReport{} o error, según firma */ fmt.Errorf("rep inode: -ruta debe ser numérica o '/': %v", err)
		}
		inIdx = int32(n)
	}

	if inIdx < 0 || inIdx >= sb.SInodesCount {
		return fmt.Errorf("rep inode: índice fuera de rango (0..%d): %d", sb.SInodesCount-1, inIdx)
	}

	ino, err := readInodeAt(mp, sb, inIdx)
	if err != nil {
		return fmt.Errorf("rep inode: leyendo inodo %d: %w", inIdx, err)
	}

	rep := buildInodeReport(mp, id, inIdx, ino)

	finalPath, format := resolveOutPathInode(outPath, id, inIdx)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep inode: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_INODE(finalPath, rep)
	default:
		return fmt.Errorf("rep inode: formato no soportado")
	}
}

func readSuperBlock(mp *mount.MountedPartition) (ext2.SuperBloque, error) {
	var sb ext2.SuperBloque
	f, err := os.Open(mp.DiskPath)
	if err != nil {
		return sb, err
	}
	defer f.Close()
	if _, err := f.Seek(mp.Start, io.SeekStart); err != nil {
		return sb, err
	}
	if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
		return sb, err
	}
	return sb, nil
}

func readInodeAt(mp *mount.MountedPartition, sb ext2.SuperBloque, idx int32) (ext2.Inodo, error) {
	var ino ext2.Inodo
	off := mp.Start + sb.SInodeStart + int64(idx)*int64(sb.SInodeS)
	f, err := os.Open(mp.DiskPath)
	if err != nil {
		return ino, err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return ino, err
	}
	if err := binary.Read(f, binary.LittleEndian, &ino); err != nil {
		return ino, err
	}
	return ino, nil
}

// ---------- Transformación a JSON ----------

func buildInodeReport(mp *mount.MountedPartition, id string, idx int32, ino ext2.Inodo) InodeReport {
	rep := InodeReport{
		Kind:     "inode",
		DiskPath: mp.DiskPath,
		ID:       id,
		Index:    idx,
		Type:     decodeType(ino.IType),
		RawType:  ino.IType,
		Size:     ino.ISize,
		UID:      ino.IUid,
		GID:      ino.IGid,
		Perm:     decodePerm(ino.IPerm[:]),
		PermRaw:  bytesCopy(ino.IPerm[:]),
		ATime:    toRFC3339(ino.IAtime),
		MTime:    toRFC3339(ino.IMtime),
		CTime:    toRFC3339(ino.ICtime),
	}

	blocks := make([]int32, 0, len(ino.IBlock))
	for _, b := range ino.IBlock {
		if b >= 0 {
			blocks = append(blocks, b)
		}
	}
	rep.Blocks = compactBlocks32(rep.Blocks)
	rep.BlocksUsed = len(rep.Blocks)

	return rep
}

func compactBlocks32(in []int32) []int32 {
	m := make(map[int32]struct{}, len(in))
	out := make([]int32, 0, len(in))
	for _, b := range in {
		if b > 0 {
			if _, ok := m[b]; !ok {
				m[b] = struct{}{}
				out = append(out, b)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func decodeType(t byte) string {
	switch t {
	case 1:
		return "file"
	case 2:
		return "dir"
	default:
		return "unknown"
	}
}

func decodePerm(p []byte) string {
	i := len(p)
	for i > 0 && (p[i-1] == 0 || p[i-1] == ' ') {
		i--
	}
	s := string(p[:i])
	s = strings.TrimSpace(s)
	if s == "" {
		return "000"
	}
	return s
}

func bytesCopy(p []byte) []byte {
	out := make([]byte, len(p))
	copy(out, p)
	return out
}

func toRFC3339(epoch int64) string {
	if epoch <= 0 {
		return ""
	}
	return time.Unix(epoch, 0).Format(time.RFC3339)
}

// ---------- Salidas ----------

func resolveOutPathInode(out string, id string, idx int32) (string, string) {
	out = strings.TrimSpace(out)
	base := fmt.Sprintf("inode_%s_%d.json", id, idx)
	if out == "" {
		return base, "json"
	}
	ext := strings.ToLower(filepath.Ext(out))
	if ext == ".json" {
		return out, "json"
	}
	if ext == ".html" || ext == ".htm" {
		return out, "html"
	}
	st, err := os.Stat(out)
	if err == nil && st.IsDir() {
		return filepath.Join(out, base), "json"
	}
	if ext == "" {
		return out + ".json", "json"
	}
	return out, "json"
}

func writeHTML_INODE(path string, rep InodeReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>INODE Report</title>")
	b.WriteString(`<style>body{font-family:system-ui,Segoe UI,Roboto,Arial}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.4rem .6rem}th{background:#f5f5f5}</style>`)
	b.WriteString("<h2>INODE</h2>")
	fmt.Fprintf(&b, "<p><b>Disco:</b> %s<br><b>ID:</b> %s<br><b>Inodo #</b> %d<br><b>Tipo:</b> %s (raw=%d)<br><b>Tamaño:</b> %d<br><b>UID/GID:</b> %d/%d<br><b>Perm:</b> %s</p>",
		escape(rep.DiskPath), escape(rep.ID), rep.Index, escape(rep.Type), rep.RawType, rep.Size, rep.UID, rep.GID, escape(rep.Perm))

	b.WriteString("<table><thead><tr><th>Bloques (IBlock)</th></tr></thead><tbody><tr><td>")
	if len(rep.Blocks) == 0 {
		b.WriteString("(sin punteros)")
	} else {
		txt := make([]string, 0, len(rep.Blocks))
		for _, v := range rep.Blocks {
			txt = append(txt, fmt.Sprintf("%d", v))
		}
		b.WriteString(escape(strings.Join(txt, ", ")))
	}
	b.WriteString("</td></tr></tbody></table>")

	b.WriteString("<h3>Tiempos</h3><table><tbody>")
	fmt.Fprintf(&b, "<tr><th>ATime</th><td>%s</td></tr>", escape(rep.ATime))
	fmt.Fprintf(&b, "<tr><th>MTime</th><td>%s</td></tr>", escape(rep.MTime))
	fmt.Fprintf(&b, "<tr><th>CTime</th><td>%s</td></tr>", escape(rep.CTime))
	b.WriteString("</tbody></table>")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func BuildInode(reg *mount.Registry, id string, ruta string) (InodeReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return InodeReport{}, fmt.Errorf("rep inode: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return InodeReport{}, fmt.Errorf("rep inode: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return InodeReport{}, fmt.Errorf("rep inode: leyendo super bloque: %w", err)
	}
	if sb.SInodesCount <= 0 {
		return InodeReport{}, fmt.Errorf("rep inode: la partición con id %s no está formateada (SInodesCount=0). Ejecute mkfs -id=%s", id, id)
	}

	ruta = strings.TrimSpace(ruta)
	var inIdx int32
	if ruta == "" || ruta == "/" || strings.EqualFold(ruta, "root") {

		inIdx = 0
	} else {
		n, err := strconv.ParseInt(ruta, 10, 32)
		if err != nil {
			return InodeReport{}, fmt.Errorf("rep inode: -ruta debe ser numérica o '/': %v", err)
		}
		inIdx = int32(n)
	}

	if inIdx < 0 || inIdx >= sb.SInodesCount {
		return InodeReport{}, fmt.Errorf("rep inode: índice fuera de rango (0..%d): %d", sb.SInodesCount-1, inIdx)
	}

	ino, err := readInodeAt(mp, sb, inIdx)
	if err != nil {
		return InodeReport{}, fmt.Errorf("rep inode: leyendo inodo %d: %w", inIdx, err)
	}
	return buildInodeReport(mp, id, inIdx, ino), nil
}
