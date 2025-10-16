package reports

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/diskio"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/structs"
)

// --------- Tipos de salida  ---------

type MBRReport struct {
	Kind      string          `json:"kind"`
	DiskPath  string          `json:"diskPath"`
	Created   string          `json:"created"`
	Size      int64           `json:"sizeBytes"`
	Signature int64           `json:"signature"`
	Fit       string          `json:"fit"`
	RawFit    byte            `json:"rawFit"`
	Parts     []MBRPartReport `json:"partitions"`
}

type MBRPartReport struct {
	Index       int    `json:"index"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Fit         string `json:"fit"`
	RawStatus   byte   `json:"rawStatus"`
	RawType     byte   `json:"rawType"`
	RawFit      byte   `json:"rawFit"`
	Start       int64  `json:"start"`
	Size        int64  `json:"size"`
	Name        string `json:"name"`
	ID          string `json:"id,omitempty"`
	Correlative int    `json:"correlative,omitempty"`
	Usable      bool   `json:"usable"`
}

func readEBRAt(diskPath string, offset int64) (structs.EBR, error) {
	f, err := os.Open(diskPath)
	if err != nil {
		return structs.EBR{}, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return structs.EBR{}, err
	}

	var ebr structs.EBR
	if err := binary.Read(f, binary.LittleEndian, &ebr); err != nil {
		return structs.EBR{}, err
	}
	return ebr, nil
}

func appendLogicalFromEBR(rep *MBRReport, diskPath string, extStart int64) error {
	if extStart <= 0 {
		return fmt.Errorf("extendida con start inválido: %d", extStart)
	}

	const maxEBRs = 128
	off := extStart

	for i := 0; i < maxEBRs; i++ {
		ebr, err := readEBRAt(diskPath, off)
		if err != nil {
			if i == 0 {

				return err
			}

			return nil
		}

		if ebr.Part_s <= 0 {
			return nil
		}

		logicalStart := extStart + ebr.Part_start

		rep.Parts = append(rep.Parts, MBRPartReport{
			Index:     len(rep.Parts),
			Status:    string([]byte{ebr.Part_status}),
			Type:      "l",
			Fit:       mapFit(ebr.Part_fit),
			RawStatus: ebr.Part_status,
			RawType:   'l',
			RawFit:    ebr.Part_fit,
			Start:     logicalStart,
			Size:      ebr.Part_s,
			Name:      trimName(ebr.Part_name[:]),
			Usable:    (ebr.Part_s > 0 && logicalStart >= 0),
		})

		if ebr.Part_next <= 0 {
			return nil
		}

		off = extStart + ebr.Part_next
	}

	return fmt.Errorf("cadena EBR demasiado larga o con ciclo")
}

func BuildMBR(reg *mount.Registry, id string) (MBRReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MBRReport{}, fmt.Errorf("rep mbr: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return MBRReport{}, fmt.Errorf("rep mbr: id %q no está montado", id)
	}

	mbr, err := diskio.ReadMBR(mp.DiskPath)
	if err != nil {
		return MBRReport{}, fmt.Errorf("rep mbr: leyendo MBR: %w", err)
	}

	rep := MBRReport{
		Kind:      "mbr",
		DiskPath:  mp.DiskPath,
		Created:   time.Unix(mbr.Mbr_fecha_creacion, 0).Format(time.RFC3339),
		Size:      mbr.Mbr_tamano,
		Signature: mbr.Mbr_dsk_signature,
		Fit:       mapFit(mbr.Dsk_fit),
		RawFit:    mbr.Dsk_fit,
	}

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		p := mbr.Mbr_partitions[i]
		pr := MBRPartReport{
			Index:       i,
			Status:      string([]byte{p.Part_status}),
			Type:        string([]byte{p.Part_type}),
			Fit:         mapFit(p.Part_fit),
			RawStatus:   p.Part_status,
			RawType:     p.Part_type,
			RawFit:      p.Part_fit,
			Start:       p.Part_start,
			Size:        p.Part_s,
			Name:        trimName(p.Part_name[:]),
			Usable:      (p.Part_s > 0 && p.Part_start >= 0),
			ID:          trimName(p.Part_id[:]),
			Correlative: int(p.Part_correlative),
		}
		rep.Parts = append(rep.Parts, pr)

		if p.Part_type == 'e' || p.Part_type == 'E' {
			if err := appendLogicalFromEBR(&rep, mp.DiskPath, p.Part_start); err != nil {
				fmt.Println("WARN: leyendo EBR:", err)
			}
		}
	}

	return rep, nil
}

func GenerateMBR(reg *mount.Registry, id, outPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("rep mbr: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rep mbr: id %q no está montado", id)
	}

	mbr, err := diskio.ReadMBR(mp.DiskPath)
	if err != nil {
		return fmt.Errorf("rep mbr: leyendo MBR: %w", err)
	}

	rep := MBRReport{
		Kind:      "mbr",
		DiskPath:  mp.DiskPath,
		Created:   time.Unix(mbr.Mbr_fecha_creacion, 0).Format(time.RFC3339),
		Size:      mbr.Mbr_tamano,
		Signature: mbr.Mbr_dsk_signature,
		Fit:       mapFit(mbr.Dsk_fit),
		RawFit:    mbr.Dsk_fit,
	}

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		p := mbr.Mbr_partitions[i]
		pr := MBRPartReport{
			Index:     i,
			Status:    string([]byte{p.Part_status}),
			Type:      string([]byte{p.Part_type}),
			Fit:       mapFit(p.Part_fit),
			RawStatus: p.Part_status,
			RawType:   p.Part_type,
			RawFit:    p.Part_fit,
			Start:     p.Part_start,
			Size:      p.Part_s,
			Name:      trimName(p.Part_name[:]),
			Usable:    (p.Part_s > 0 && p.Part_start >= 0),
		}

		if _, has := any(p).(structs.Partition); has {

			pr.ID = trimName(p.Part_id[:])
			pr.Correlative = int(p.Part_correlative)
		}
		pr.ID = trimName(p.Part_id[:])
		pr.Correlative = int(p.Part_correlative)

		rep.Parts = append(rep.Parts, pr)

		if p.Part_type == 'e' || p.Part_type == 'E' {
			if err := appendLogicalFromEBR(&rep, mp.DiskPath, p.Part_start); err != nil {
				fmt.Println("WARN: leyendo EBR:", err)
			}
		}

	}

	finalPath, format := resolveOutPath(outPath, id)

	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep mbr: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_MBR(finalPath, rep)
	default:
		return fmt.Errorf("rep mbr: formato no soportado")
	}
}

// ---------------- Helpers de salida ----------------

func writeJSON(path string, v any) error {
	low := strings.ToLower(strings.TrimSpace(path))
	if strings.HasSuffix(low, ".mia") {
		return fmt.Errorf("ruta de salida apunta a un .mia; se aborta para no sobrescribir el disco")
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeHTML_MBR(path string, rep MBRReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>MBR Report</title>")
	b.WriteString(`<style>body{font-family:system-ui,Segoe UI,Roboto,Arial}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.4rem .6rem}th{background:#f5f5f5}</style>`)
	b.WriteString("<h2>MBR</h2>")
	fmt.Fprintf(&b, "<p><b>Disk:</b> %s<br><b>Created:</b> %s<br><b>Size:</b> %d bytes<br><b>Signature:</b> %d<br><b>Fit:</b> %s (0x%02X)</p>",
		escape(rep.DiskPath), escape(rep.Created), rep.Size, rep.Signature, escape(rep.Fit), rep.RawFit)

	b.WriteString("<table><thead><tr>")
	b.WriteString("<th>#</th><th>Status</th><th>Type</th><th>Fit</th><th>Start</th><th>Size</th><th>Name</th><th>Usable</th>")
	b.WriteString("</tr></thead><tbody>")
	for _, p := range rep.Parts {
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s (0x%02X)</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%t</td></tr>",
			p.Index, escape(p.Status), p.RawStatus, escape(p.Type), escape(p.Fit), p.Start, p.Size, escape(p.Name), p.Usable)
	}
	b.WriteString("</tbody></table>")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func escape(s string) string {
	repl := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return repl.Replace(s)
}

// ---------------- Helpers de formato ----------------

func resolveOutPath(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	if out == "" {
		return fmt.Sprintf("mbr_%s.json", id), "json"
	}
	ext := strings.ToLower(filepath.Ext(out))

	if ext == ".json" {
		return out, "json"
	}
	if ext == ".html" || ext == ".htm" {
		return out, "html"
	}

	if st, err := os.Stat(out); err == nil && st.IsDir() {
		return filepath.Join(out, fmt.Sprintf("mbr_%s.json", id)), "json"
	}

	if st, err := os.Stat(out); err == nil && !st.IsDir() {
		base := strings.TrimSuffix(filepath.Base(out), ext)
		return filepath.Join(filepath.Dir(out), base+".json"), "json"
	}

	if ext == "" {

		return out + ".json", "json"
	}

	base := strings.TrimSuffix(out, ext)
	return base + ".json", "json"
}

func mapFit(b byte) string {
	switch b {
	case 'b', 'B':
		return "BF"
	case 'f', 'F':
		return "FF"
	case 'w', 'W':
		return "WF"
	default:
		return ""
	}
}

func trimName(arr []byte) string {
	i := len(arr)
	for i > 0 && (arr[i-1] == 0 || arr[i-1] == ' ') {
		i--
	}
	return string(arr[:i])
}
