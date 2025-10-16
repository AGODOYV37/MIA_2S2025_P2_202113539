package reports

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/diskio"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
)

type DiskReport struct {
	Kind     string        `json:"kind"`
	DiskPath string        `json:"diskPath"`
	Size     int64         `json:"sizeBytes"`
	MBRSize  int64         `json:"mbrBytes"`
	Segments []DiskSegment `json:"segments"`
	Extended *ExtendedView `json:"extended,omitempty"`
}

type DiskSegment struct {
	Kind    string  `json:"kind"`
	Label   string  `json:"label"`
	Start   int64   `json:"start"`
	Size    int64   `json:"size"`
	End     int64   `json:"end"`
	Percent float64 `json:"percent"`
}

type ExtendedView struct {
	Start    int64         `json:"start"`
	Size     int64         `json:"size"`
	Segments []DiskSegment `json:"segments"`
}

func GenerateDisk(reg *mount.Registry, id, outPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("rep disk: -id requerido")
	}

	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rep disk: id %q no está montado", id)
	}

	mbr, err := diskio.ReadMBR(mp.DiskPath)
	if err != nil {
		return fmt.Errorf("rep disk: leyendo MBR: %w", err)
	}

	var m structs.MBR
	mbrSize := int64(binary.Size(m))
	if mbrSize <= 0 {
		mbrSize = 512
	}

	total := mbr.Mbr_tamano

	if total <= 0 {
		if st, err := os.Stat(mp.DiskPath); err == nil {
			total = st.Size()
		} else {
			return fmt.Errorf("rep disk: leyendo MBR: tamaño total inválido y no se pudo stat: %w", err)
		}
	}

	rep := DiskReport{
		Kind:     "disk",
		DiskPath: mp.DiskPath,
		Size:     total,
		MBRSize:  mbrSize,
	}

	topSegs := []DiskSegment{
		makeSeg("MBR", "MBR", 0, mbrSize, total),
	}

	type pseg struct {
		kind  string
		label string
		start int64
		size  int64
	}
	var prim []pseg
	var ext *pseg

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_s <= 0 || p.Part_start < 0 {
			continue
		}
		name := trimName(p.Part_name[:])
		if name == "" {
			name = fmt.Sprintf("part%d", i+1)
		}
		switch p.Part_type {
		case 'p', 'P':
			prim = append(prim, pseg{"P", name, p.Part_start, p.Part_s})
		case 'e', 'E':
			tmp := pseg{"E", "extendida", p.Part_start, p.Part_s}
			ext = &tmp

		default:

			prim = append(prim, pseg{"P", name, p.Part_start, p.Part_s})
		}
	}

	for _, x := range prim {
		topSegs = append(topSegs, makeSeg(x.kind, x.label, x.start, x.size, total))
	}
	if ext != nil {
		topSegs = append(topSegs, makeSeg(ext.kind, ext.label, ext.start, ext.size, total))
	}

	topSegs = fillTopLevelFree(rep.Size, topSegs)

	sort.Slice(topSegs, func(i, j int) bool { return topSegs[i].Start < topSegs[j].Start })
	rep.Segments = normalizePercents(topSegs, total, 2)

	if ext != nil {
		extView, err := buildExtendedView(mp.DiskPath, ext.start, ext.size, total)
		if err != nil {
			fmt.Printf("rep disk: WARN extendida: %v\n", err)
		} else {
			rep.Extended = &extView
		}
	}

	finalPath, format := resolveOutPathDisk(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep disk: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_DISK(finalPath, rep)
	default:
		return fmt.Errorf("rep disk: formato no soportado")
	}
}

// ================= helpers de cálculo =================

func makeSeg(kind, label string, start, size, total int64) DiskSegment {
	if start < 0 {
		start = 0
	}
	if size < 0 {
		size = 0
	}
	seg := DiskSegment{
		Kind:  kind,
		Label: label,
		Start: start,
		Size:  size,
		End:   start + size,
	}
	if total > 0 && size > 0 {
		seg.Percent = (float64(size) / float64(total)) * 100.0
	}
	return seg
}

func fillTopLevelFree(total int64, segs []DiskSegment) []DiskSegment {

	if len(segs) == 0 {
		return []DiskSegment{makeSeg("FREE", "libre", 0, total, total)}
	}

	sort.Slice(segs, func(i, j int) bool { return segs[i].Start < segs[j].Start })

	out := make([]DiskSegment, 0, len(segs)+2)
	cur := int64(0)

	for _, s := range segs {
		if s.Start > cur {

			out = append(out, makeSeg("FREE", "libre", cur, s.Start-cur, total))
		}

		out = append(out, s)
		if s.End > cur {
			cur = s.End
		}
	}

	if total > cur {
		out = append(out, makeSeg("FREE", "libre", cur, total-cur, total))
	}
	return out
}

func buildExtendedView(diskPath string, extStart, extSize, total int64) (ExtendedView, error) {
	if extSize <= 0 {
		return ExtendedView{}, fmt.Errorf("extendida con tamaño inválido")
	}

	var ev ExtendedView
	ev.Start = extStart
	ev.Size = extSize

	var ebrZero structs.EBR
	ebrSize := int64(binary.Size(ebrZero))
	if ebrSize <= 0 {
		ebrSize = 512
	}

	const maxEBRs = 128
	off := extStart

	type inner struct {
		seg DiskSegment
	}
	var occupied []inner

	for i := 0; i < maxEBRs; i++ {
		ebr, err := readEBRAt(diskPath, off)
		if err != nil {
			if i == 0 {

				break
			}

			break
		}

		ebrSeg := makeSeg("EBR", "ebr", off, ebrSize, total)
		occupied = append(occupied, inner{ebrSeg})

		if ebr.Part_s > 0 {
			logicalStart := extStart + ebr.Part_start
			label := trimName(ebr.Part_name[:])
			if label == "" {
				label = "logica"
			}
			lSeg := makeSeg("L", label, logicalStart, ebr.Part_s, total)
			occupied = append(occupied, inner{lSeg})
		}

		if ebr.Part_next <= 0 {
			break
		}
		off = extStart + ebr.Part_next
	}

	if len(occupied) == 0 {
		return ev, nil
	}

	sort.Slice(occupied, func(i, j int) bool { return occupied[i].seg.Start < occupied[j].seg.Start })

	cur := extStart
	for _, it := range occupied {
		s := it.seg
		if s.Start > cur {
			ev.Segments = append(ev.Segments, makeSeg("FREE", "libre", cur, s.Start-cur, total))
		}
		ev.Segments = append(ev.Segments, s)
		if s.End > cur {
			cur = s.End
		}
	}

	extEnd := extStart + extSize
	if extEnd > cur {
		ev.Segments = append(ev.Segments, makeSeg("FREE", "libre", cur, extEnd-cur, total))
	}

	ev.Segments = normalizePercents(ev.Segments, total, 2)

	return ev, nil
}

// ================= helpers de salida =================

func resolveOutPathDisk(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	if out == "" {
		return fmt.Sprintf("disk_%s.json", id), "json"
	}
	ext := strings.ToLower(filepath.Ext(out))

	if ext == ".json" {
		return out, "json"
	}
	if ext == ".html" || ext == ".htm" {
		return out, "html"
	}

	if st, err := os.Stat(out); err == nil && st.IsDir() {
		return filepath.Join(out, fmt.Sprintf("disk_%s.json", id)), "json"
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

func writeHTML_DISK(path string, rep DiskReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>DISK Report</title>")
	b.WriteString(`<style>
body{font-family:system-ui,Segoe UI,Roboto,Arial;margin:16px}
h2{margin:8px 0}
.wrap{border:1px solid #ccc;border-radius:6px;overflow:hidden}
.bar{display:flex;height:28px;margin-bottom:12px}
.seg{height:28px;display:inline-block;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;font-size:12px;line-height:28px;text-align:center;color:#111;border-right:1px solid #fff;padding:0 6px}
.seg.MBR{background:#ffd166}
.seg.P{background:#06d6a0}
.seg.E{background:#118ab2;color:#fff}
.seg.FREE{background:#efefef}
.table{border-collapse:collapse;margin-top:8px}
.table th,.table td{border:1px solid #ccc;padding:.35rem .5rem}
.table th{background:#f7f7f7}
.small{color:#555;font-size:12px}
</style>`)
	b.WriteString("<h2>DISK</h2>")
	fmt.Fprintf(&b, `<p class="small"><b>Disco:</b> %s &nbsp; <b>Tamaño:</b> %d bytes &nbsp; <b>MBR:</b> %d bytes</p>`,
		escape(rep.DiskPath), rep.Size, rep.MBRSize)

	b.WriteString(`<div class="wrap"><div class="bar">`)
	for _, s := range rep.Segments {
		w := s.Percent
		if w < 0 {
			w = 0
		}
		style := fmt.Sprintf(`style="width:%.2f%%"`, w)
		cls := s.Kind
		if cls == "L" || cls == "EBR" {

			continue
		}
		label := s.Label
		if label == "" {
			label = s.Kind
		}
		fmt.Fprintf(&b, `<div class="seg %s" %s>%s</div>`, cls, style, escape(label))
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<table class="table"><thead><tr><th>Kind</th><th>Label</th><th>Start</th><th>Size</th><th>End</th><th>%</th></tr></thead><tbody>`)
	for _, s := range rep.Segments {
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%.2f</td></tr>`,
			escape(s.Kind), escape(s.Label), s.Start, s.Size, s.End, s.Percent)
	}
	b.WriteString(`</tbody></table>`)

	if rep.Extended != nil {
		b.WriteString("<h3>Extendida (detalle)</h3>")
		fmt.Fprintf(&b, `<p class="small"><b>Start:</b> %d &nbsp; <b>Size:</b> %d</p>`, rep.Extended.Start, rep.Extended.Size)

		b.WriteString(`<table class="table"><thead><tr><th>Kind</th><th>Label</th><th>Start</th><th>Size</th><th>End</th><th>% del disco</th></tr></thead><tbody>`)
		for _, s := range rep.Extended.Segments {
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%.2f</td></tr>`,
				escape(s.Kind), escape(s.Label), s.Start, s.Size, s.End, s.Percent)
		}
		b.WriteString(`</tbody></table>`)
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func BuildDisk(reg *mount.Registry, id string) (DiskReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return DiskReport{}, fmt.Errorf("rep disk: -id requerido")
	}

	mp, ok := reg.GetByID(id)
	if !ok {
		return DiskReport{}, fmt.Errorf("rep disk: id %q no está montado", id)
	}

	mbr, err := diskio.ReadMBR(mp.DiskPath)
	if err != nil {
		return DiskReport{}, fmt.Errorf("rep disk: leyendo MBR: %w", err)
	}

	var m structs.MBR
	mbrSize := int64(binary.Size(m))
	if mbrSize <= 0 {
		mbrSize = 512
	}

	total := mbr.Mbr_tamano

	if total <= 0 {
		if st, err2 := os.Stat(mp.DiskPath); err2 == nil {
			total = st.Size()
		} else {
			return DiskReport{}, fmt.Errorf("rep disk: leyendo MBR: tamaño total inválido y no se pudo stat: %w", err2)
		}
	}

	rep := DiskReport{
		Kind:     "disk",
		DiskPath: mp.DiskPath,
		Size:     total,
		MBRSize:  mbrSize,
	}

	topSegs := []DiskSegment{
		makeSeg("MBR", "MBR", 0, mbrSize, total),
	}

	type pseg struct {
		kind  string
		label string
		start int64
		size  int64
	}
	var prim []pseg
	var ext *pseg

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_s <= 0 || p.Part_start < 0 {
			continue
		}
		name := trimName(p.Part_name[:])
		if name == "" {
			name = fmt.Sprintf("part%d", i+1)
		}
		switch p.Part_type {
		case 'p', 'P':
			prim = append(prim, pseg{"P", name, p.Part_start, p.Part_s})
		case 'e', 'E':
			tmp := pseg{"E", "extendida", p.Part_start, p.Part_s}
			ext = &tmp
		default:

			prim = append(prim, pseg{"P", name, p.Part_start, p.Part_s})
		}
	}

	for _, x := range prim {
		topSegs = append(topSegs, makeSeg(x.kind, x.label, x.start, x.size, total))
	}
	if ext != nil {
		topSegs = append(topSegs, makeSeg(ext.kind, ext.label, ext.start, ext.size, total))
	}

	topSegs = fillTopLevelFree(rep.Size, topSegs)

	sort.Slice(topSegs, func(i, j int) bool { return topSegs[i].Start < topSegs[j].Start })
	rep.Segments = normalizePercents(topSegs, total, 2)

	if ext != nil {
		extView, err := buildExtendedView(mp.DiskPath, ext.start, ext.size, total)
		if err != nil {
			fmt.Printf("rep disk: WARN extendida: %v\n", err)
		} else {
			rep.Extended = &extView
		}
	}

	return rep, nil
}

func normalizePercents(segs []DiskSegment, total int64, decimals int) []DiskSegment {
	if total <= 0 || len(segs) == 0 {
		return segs
	}
	scale := 1
	for i := 0; i < decimals; i++ {
		scale *= 10
	}
	target := 100 * scale

	type frac struct {
		idx int
		rem float64
	}

	out := make([]DiskSegment, len(segs))
	copy(out, segs)

	floors := make([]int, len(segs))
	rems := make([]frac, 0, len(segs))
	sum := 0

	for i, s := range segs {
		if s.Size <= 0 {
			floors[i] = 0
			continue
		}
		exact := (float64(s.Size) / float64(total)) * float64(target)
		f := int(math.Floor(exact + 1e-9))
		floors[i] = f
		sum += f
		rems = append(rems, frac{idx: i, rem: exact - float64(f)})
	}

	diff := target - sum

	sort.Slice(rems, func(i, j int) bool { return rems[i].rem > rems[j].rem })
	for k := 0; diff > 0 && k < len(rems); k++ {
		floors[rems[k].idx]++
		diff--
		if k == len(rems)-1 && diff > 0 {
			k = -1
		}
	}

	if diff < 0 {
		sort.Slice(rems, func(i, j int) bool { return rems[i].rem < rems[j].rem })
		for k := 0; diff < 0 && k < len(rems); k++ {
			if floors[rems[k].idx] > 0 {
				floors[rems[k].idx]--
				diff++
			}
			if k == len(rems)-1 && diff < 0 {
				k = -1
			}
		}
	}

	for i := range out {
		out[i].Percent = float64(floors[i]) / float64(scale)
	}
	return out
}
