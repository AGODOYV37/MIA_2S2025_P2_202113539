package reports

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

type InodeMini struct {
	Index      int32  `json:"index"`
	Type       string `json:"type"`
	RawType    byte   `json:"rawType"`
	Size       int64  `json:"size"`
	BlocksUsed int32  `json:"blocksUsed"`
}

type InodesReport struct {
	Kind     string      `json:"kind"`
	DiskPath string      `json:"diskPath"`
	ID       string      `json:"id"`
	Count    int32       `json:"count"`
	Items    []InodeMini `json:"items"`
}

func BuildInodes(reg *mount.Registry, id string, maxItems int) (InodesReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return InodesReport{}, fmt.Errorf("rep inodes: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return InodesReport{}, fmt.Errorf("rep inodes: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return InodesReport{}, fmt.Errorf("rep inodes: leyendo super bloque: %w", err)
	}
	if sb.SInodesCount <= 0 {
		return InodesReport{}, fmt.Errorf("rep inodes: partición no formateada; ejecute mkfs")
	}

	out := InodesReport{
		Kind:     "inodes",
		DiskPath: mp.DiskPath,
		ID:       id,
		Count:    sb.SInodesCount,
	}

	collected := make([]InodeMini, 0, 128)
	for idx := int32(0); idx < sb.SInodesCount; idx++ {
		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			continue
		}
		mini := InodeMini{
			Index:      idx,
			Type:       decodeType(ino.IType),
			RawType:    ino.IType,
			Size:       int64(ino.ISize),
			BlocksUsed: countBlocks32(ino),
		}

		if mini.BlocksUsed > 0 || strings.EqualFold(mini.Type, "dir") {
			collected = append(collected, mini)
			if maxItems > 0 && len(collected) >= maxItems {
				break
			}
		}
	}
	sort.Slice(collected, func(i, j int) bool { return collected[i].Index < collected[j].Index })
	out.Items = collected
	return out, nil
}

func countBlocks32(in ext2.Inodo) int32 {
	var used int32
	for _, b := range in.IBlock {
		if b > 0 {
			used++
		}
	}
	return used
}

func GenerateInodes(reg *mount.Registry, id, outPath string) error {
	rep, err := BuildInodes(reg, id, 0)
	if err != nil {
		return err
	}
	finalPath, format := resolveOutPathInodes(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep inodes: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_INODES(finalPath, rep)
	default:
		return fmt.Errorf("rep inodes: formato no soportado")
	}
}

func resolveOutPathInodes(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	if out == "" {
		return fmt.Sprintf("inodes_%s.json", id), "json"
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
		return filepath.Join(out, fmt.Sprintf("inodes_%s.json", id)), "json"
	}
	if ext == "" {
		return out + ".json", "json"
	}
	return out, "json"
}

func writeHTML_INODES(path string, rep InodesReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>INODES Report</title>")
	b.WriteString(`<style>
body{font-family:system-ui,Segoe UI,Roboto,Arial;margin:16px}
h2{margin:8px 0}
table{border-collapse:collapse;width:100%}
th,td{border:1px solid #ccc;padding:.4rem .6rem}
th{background:#f7f7f7}
.small{color:#555;font-size:12px}
</style>`)
	fmt.Fprintf(&b, "<h2>INODES</h2>")
	fmt.Fprintf(&b, `<p class="small"><b>Disco:</b> %s &nbsp; <b>Partición:</b> %s &nbsp; <b>Inodos totales:</b> %d</p>`,
		escape(rep.DiskPath), escape(rep.ID), rep.Count)

	b.WriteString("<table><thead><tr>")
	b.WriteString("<th>#</th><th>Tipo</th><th>Raw</th><th>Tamaño</th><th>Bloques usados</th>")
	b.WriteString("</tr></thead><tbody>")
	for _, it := range rep.Items {
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td></tr>",
			it.Index, escape(it.Type), it.RawType, it.Size, it.BlocksUsed)
	}
	b.WriteString("</tbody></table>")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
