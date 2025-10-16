package reports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// ===================== Modelo =====================

type SBReport struct {
	Kind     string `json:"kind"`
	DiskPath string `json:"diskPath"`
	ID       string `json:"id"`

	BlockSize       int32 `json:"blockSize"`
	InodesCount     int32 `json:"inodesCount"`
	BlocksCount     int32 `json:"blocksCount"`
	FreeInodes      int32 `json:"freeInodes"`
	FreeBlocks      int32 `json:"freeBlocks"`
	InodeSize       int32 `json:"inodeSize"`
	BmInodeStart    int64 `json:"bmInodeStart"`
	BmBlockStart    int64 `json:"bmBlockStart"`
	InodeTableStart int64 `json:"inodeTableStart"`
	BlockStart      int64 `json:"blockStart"`

	BitmapUsedInodes int `json:"bitmapUsedInodes"`
	BitmapFreeInodes int `json:"bitmapFreeInodes"`
	BitmapUsedBlocks int `json:"bitmapUsedBlocks"`
	BitmapFreeBlocks int `json:"bitmapFreeBlocks"`
}

// ===================== Build / Generate =====================

func BuildSB(reg *mount.Registry, id string) (SBReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SBReport{}, fmt.Errorf("rep sb: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return SBReport{}, fmt.Errorf("rep sb: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return SBReport{}, fmt.Errorf("rep sb: leyendo super bloque: %w", err)
	}

	bmIn, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return SBReport{}, fmt.Errorf("rep sb: cargando bitmaps: %w", err)
	}

	usedIn, usedBl := 0, 0
	for i := int32(0); i < sb.SInodesCount && int(i) < len(bmIn); i++ {
		if bmIn[i] != 0 {
			usedIn++
		}
	}
	for i := int32(0); i < sb.SBlocksCount && int(i) < len(bmBl); i++ {
		if bmBl[i] != 0 {
			usedBl++
		}
	}
	freeIn := int(sb.SInodesCount) - usedIn
	if freeIn < 0 {
		freeIn = 0
	}
	freeBl := int(sb.SBlocksCount) - usedBl
	if freeBl < 0 {
		freeBl = 0
	}

	rep := SBReport{
		Kind:      "sb",
		DiskPath:  mp.DiskPath,
		ID:        id,
		BlockSize: ext2.BlockSize,

		InodesCount:     sb.SInodesCount,
		BlocksCount:     sb.SBlocksCount,
		InodeSize:       sb.SInodeS,
		BmInodeStart:    sb.SBmInodeStart,
		BmBlockStart:    sb.SBmBlockStart,
		InodeTableStart: sb.SInodeStart,
		BlockStart:      sb.SBlockStart,

		BitmapUsedInodes: usedIn,
		BitmapFreeInodes: freeIn,
		BitmapUsedBlocks: usedBl,
		BitmapFreeBlocks: freeBl,
	}

	rep.FreeInodes = int32(freeIn)
	rep.FreeBlocks = int32(freeBl)

	return rep, nil
}

func GenerateSB(reg *mount.Registry, id, outPath string) error {
	rep, err := BuildSB(reg, id)
	if err != nil {
		return err
	}
	finalPath, format := resolveOutPathSB(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep sb: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_SB(finalPath, rep)
	default:
		return fmt.Errorf("rep sb: formato no soportado")
	}
}

// ===================== Salida =====================

func resolveOutPathSB(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	base := fmt.Sprintf("sb_%s.json", id)
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
	if st, err := os.Stat(out); err == nil && st.IsDir() {
		return filepath.Join(out, base), "json"
	}
	if ext == "" {
		return out + ".json", "json"
	}
	return out, "json"
}

func writeHTML_SB(path string, rep SBReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>Superblock Report</title>")
	b.WriteString(`<style>body{font-family:system-ui,Segoe UI,Roboto,Arial}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.4rem .6rem}th{background:#f5f5f5;text-align:left}</style>`)
	b.WriteString("<h2>Superblock</h2>")
	fmt.Fprintf(&b, "<p><b>Disk:</b> %s &nbsp;|&nbsp; <b>ID:</b> %s</p>", escape(rep.DiskPath), escape(rep.ID))

	b.WriteString("<table><tbody>")
	row := func(k string, v any) {
		fmt.Fprintf(&b, "<tr><th>%s</th><td>%v</td></tr>", escape(k), v)
	}

	row("BlockSize", rep.BlockSize)
	row("InodesCount", rep.InodesCount)
	row("BlocksCount", rep.BlocksCount)
	row("FreeInodes (bitmap)", rep.BitmapFreeInodes)
	row("FreeBlocks (bitmap)", rep.BitmapFreeBlocks)
	row("UsedInodes (bitmap)", rep.BitmapUsedInodes)
	row("UsedBlocks (bitmap)", rep.BitmapUsedBlocks)
	row("InodeSize", rep.InodeSize)
	row("BmInodeStart", rep.BmInodeStart)
	row("BmBlockStart", rep.BmBlockStart)
	row("InodeTableStart", rep.InodeTableStart)
	row("BlockStart", rep.BlockStart)

	b.WriteString("</tbody></table>")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
