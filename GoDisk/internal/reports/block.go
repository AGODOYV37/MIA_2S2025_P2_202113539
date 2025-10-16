package reports

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// ---------------------- Modelo JSON ----------------------

type BlockReport struct {
	Kind      string      `json:"kind"`
	DiskPath  string      `json:"diskPath"`
	ID        string      `json:"id"`
	BlockSize int32       `json:"blockSize"`
	Count     int32       `json:"count"`
	Used      int         `json:"used"`
	Blocks    []BlockItem `json:"blocks"`
}

type BlockItem struct {
	Index    int32          `json:"index"`
	Type     string         `json:"type"`
	RefCount int            `json:"refCount"`
	Dir      *DirBlockView  `json:"dir,omitempty"`
	File     *FileBlockView `json:"file,omitempty"`
	Ptr      *PtrBlockView  `json:"ptr,omitempty"`
}

type DirBlockView struct {
	Entries []DirEntry `json:"entries"`
}

type DirEntry struct {
	Name  string `json:"name"`
	Inode int32  `json:"inode"`
}

type FileBlockView struct {
	Size    int    `json:"size"`
	Preview string `json:"preview"`
}

type PtrBlockView struct {
	Pointers []int32 `json:"pointers"`
}

// ---------------------- API Pública ----------------------

func GenerateBlock(reg *mount.Registry, id, outPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("rep block: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rep block: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return fmt.Errorf("rep block: leyendo super bloque: %w", err)
	}

	bmIn, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return fmt.Errorf("rep block: cargando bitmaps: %w", err)
	}

	refType, refCount, err := indexBlocksFromInodes(mp, sb, bmIn)
	if err != nil {
		return fmt.Errorf("rep block: indexando inodos: %w", err)
	}

	usedIdxs := make([]int32, 0, int(sb.SBlocksCount))
	for i := int32(0); i < sb.SBlocksCount; i++ {
		if bmBl[i] != 0 {
			usedIdxs = append(usedIdxs, i)
		}
	}
	sort.Slice(usedIdxs, func(i, j int) bool { return usedIdxs[i] < usedIdxs[j] })

	rep := BlockReport{
		Kind:      "block",
		DiskPath:  mp.DiskPath,
		ID:        id,
		BlockSize: ext2.BlockSize,
		Count:     sb.SBlocksCount,
		Used:      len(usedIdxs),
	}

	//  Decodificar contenido por tipo
	for _, bi := range usedIdxs {
		tp := refType[bi]
		if tp == "" {
			tp = "unknown"
		}
		cnt := refCount[bi]

		item := BlockItem{
			Index:    bi,
			Type:     tp,
			RefCount: cnt,
		}

		raw, err := readBlockBytes(mp, sb, bi)
		if err != nil {

			rep.Blocks = append(rep.Blocks, item)
			continue
		}

		switch tp {
		case "dir":
			item.Dir = &DirBlockView{Entries: parseDirEntries(raw)}
		case "file":
			size, prev := parseFilePreview(raw)
			item.File = &FileBlockView{Size: size, Preview: prev}
		case "ptr":
			item.Ptr = &PtrBlockView{Pointers: parsePointers(raw)}
		default:

			if esDirHeur(raw) {
				item.Type = "dir"
				item.Dir = &DirBlockView{Entries: parseDirEntries(raw)}
			} else {
				size, prev := parseFilePreview(raw)
				if size > 0 {
					item.Type = "file"
					item.File = &FileBlockView{Size: size, Preview: prev}
				} else {

					ps := parsePointers(raw)
					valid := 0
					for _, v := range ps {
						if v >= 0 {
							valid++
						}
					}
					if valid > 0 {
						item.Type = "ptr"
						item.Ptr = &PtrBlockView{Pointers: ps}
					}
				}
			}
		}

		rep.Blocks = append(rep.Blocks, item)
	}

	finalPath, format := resolveOutPathBlock(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep block: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		return writeHTML_BLOCK(finalPath, rep)
	default:
		return fmt.Errorf("rep block: formato no soportado")
	}
}

// ---------------------- Lectura / Bitmaps ----------------------

func loadBitmapsForReport(mp *mount.MountedPartition, sb ext2.SuperBloque) ([]byte, []byte, error) {
	szIn := int(sb.SInodesCount)
	szBl := int(sb.SBlocksCount)

	bmIn, err := readBytesAt(mp.DiskPath, mp.Start+sb.SBmInodeStart, szIn)
	if err != nil {
		return nil, nil, err
	}
	bmBl, err := readBytesAt(mp.DiskPath, mp.Start+sb.SBmBlockStart, szBl)
	if err != nil {
		return nil, nil, err
	}
	return bmIn, bmBl, nil
}

func readBlockBytes(mp *mount.MountedPartition, sb ext2.SuperBloque, blk int32) ([]byte, error) {
	off := mp.Start + sb.SBlockStart + int64(blk)*int64(ext2.BlockSize)
	return readBytesAt(mp.DiskPath, off, int(ext2.BlockSize))
}

func readBytesAt(path string, off int64, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(f, buf)
	return buf, err
}

// ---------------------- Clasificación por inodos ----------------------

func indexBlocksFromInodes(mp *mount.MountedPartition, sb ext2.SuperBloque, bmIn []byte) (map[int32]string, map[int32]int, error) {
	refType := make(map[int32]string)
	refCnt := make(map[int32]int)

	for idx := int32(0); idx < sb.SInodesCount; idx++ {
		if bmIn[idx] == 0 {
			continue
		}
		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			continue
		}
		t := decodeType(ino.IType)

		for i, b := range ino.IBlock {
			if b < 0 || b >= sb.SBlocksCount {
				continue
			}
			if i >= 12 {
				if _, ok := refType[b]; !ok {
					refType[b] = "ptr"
				}
			} else {

				if t == "dir" || t == "file" {
					refType[b] = t
				}
			}
			refCnt[b]++
		}
	}
	return refType, refCnt, nil
}

// ---------------------- Decodificadores de bloque ----------------------

func parseDirEntries(raw []byte) []DirEntry {
	out := make([]DirEntry, 0, 4)
	step := 16
	for i := 0; i+step <= len(raw) && len(out) < 4; i += step {
		nameBytes := raw[i : i+12]
		inoBytes := raw[i+12 : i+16]
		name := trimRightZerosSpaces(nameBytes)
		inode := int32(binary.LittleEndian.Uint32(inoBytes))
		if name == "" && inode < 0 {
			continue
		}
		if inode >= 0 && name != "" {
			out = append(out, DirEntry{Name: name, Inode: inode})
		}
	}
	return out
}

func parseFilePreview(raw []byte) (int, string) {
	size := usefulSize(raw)
	if size <= 0 {
		return 0, ""
	}
	data := raw[:size]
	const maxChars = 256
	var b strings.Builder
	for len(data) > 0 && b.Len() < maxChars {
		r, sz := utf8.DecodeRune(data)
		if r == utf8.RuneError && sz == 1 {
			b.WriteRune('�')
			data = data[1:]
		} else {
			b.WriteRune(r)
			data = data[sz:]
		}
	}
	return size, b.String()
}

func parsePointers(raw []byte) []int32 {
	out := make([]int32, 0, 16)
	for i := 0; i+4 <= len(raw) && len(out) < 16; i += 4 {
		v := int32(binary.LittleEndian.Uint32(raw[i : i+4]))
		if v >= 0 {
			out = append(out, v)
		}
	}
	return out
}

func esDirHeur(raw []byte) bool {
	ents := parseDirEntries(raw)
	return len(ents) > 0
}

func usefulSize(b []byte) int {
	i := len(b)
	for i > 0 && b[i-1] == 0 {
		i--
	}
	return i
}

func trimRightZerosSpaces(b []byte) string {
	i := len(b)
	for i > 0 && (b[i-1] == 0 || b[i-1] == ' ') {
		i--
	}
	return string(b[:i])
}

func resolveOutPathBlock(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	base := fmt.Sprintf("block_%s.json", id)
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

func writeHTML_BLOCK(path string, rep BlockReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>BLOCK Report</title>")
	b.WriteString(`<style>
body{font-family:system-ui,Segoe UI,Roboto,Arial;margin:16px}
table{border-collapse:collapse}
td,th{border:1px solid #ccc;padding:.4rem .6rem}
th{background:#f5f5f5}
code{background:#f7f7f7;padding:.15rem .3rem;border-radius:4px}
.small{color:#555;font-size:12px}
</style>`)
	b.WriteString("<h2>BLOCKS</h2>")
	fmt.Fprintf(&b, `<p class="small"><b>Disco:</b> %s<br><b>ID:</b> %s<br><b>BlockSize:</b> %d<br><b>Usados:</b> %d de %d</p>`,
		escape(rep.DiskPath), escape(rep.ID), rep.BlockSize, rep.Used, rep.Count)

	b.WriteString("<table><thead><tr>")
	b.WriteString("<th>Index</th><th>Type</th><th>Ref</th><th>Detalle</th>")
	b.WriteString("</tr></thead><tbody>")

	for _, it := range rep.Blocks {
		b.WriteString("<tr>")
		fmt.Fprintf(&b, "<td>%d</td><td>%s</td><td>%d</td><td>", it.Index, escape(it.Type), it.RefCount)

		switch it.Type {
		case "dir":
			if it.Dir == nil || len(it.Dir.Entries) == 0 {
				b.WriteString("(vacío)")
			} else {
				var rows []string
				for _, e := range it.Dir.Entries {
					rows = append(rows, fmt.Sprintf("%s → %d", escape(e.Name), e.Inode))
				}
				b.WriteString(escape(strings.Join(rows, " | ")))
			}
		case "file":
			if it.File == nil {
				b.WriteString("(sin preview)")
			} else {
				fmt.Fprintf(&b, "size=%d preview=<code>%s</code>", it.File.Size, escape(it.File.Preview))
			}
		case "ptr":
			if it.Ptr == nil || len(it.Ptr.Pointers) == 0 {
				b.WriteString("(sin punteros)")
			} else {
				vals := make([]string, 0, len(it.Ptr.Pointers))
				for _, v := range it.Ptr.Pointers {
					vals = append(vals, fmt.Sprintf("%d", v))
				}
				b.WriteString(escape(strings.Join(vals, ", ")))
			}
		default:
			b.WriteString("(desconocido)")
		}

		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func BuildBlock(reg *mount.Registry, id string) (BlockReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return BlockReport{}, fmt.Errorf("rep block: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return BlockReport{}, fmt.Errorf("rep block: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return BlockReport{}, fmt.Errorf("rep block: leyendo super bloque: %w", err)
	}

	bmIn, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return BlockReport{}, fmt.Errorf("rep block: cargando bitmaps: %w", err)
	}

	refType, refCount, err := indexBlocksFromInodes(mp, sb, bmIn)
	if err != nil {
		return BlockReport{}, fmt.Errorf("rep block: indexando inodos: %w", err)
	}

	usedIdxs := make([]int32, 0, int(sb.SBlocksCount))
	for i := int32(0); i < sb.SBlocksCount; i++ {
		if bmBl[i] != 0 {
			usedIdxs = append(usedIdxs, i)
		}
	}
	sort.Slice(usedIdxs, func(i, j int) bool { return usedIdxs[i] < usedIdxs[j] })

	rep := BlockReport{
		Kind:      "block",
		DiskPath:  mp.DiskPath,
		ID:        id,
		BlockSize: ext2.BlockSize,
		Count:     sb.SBlocksCount,
		Used:      len(usedIdxs),
	}

	for _, bi := range usedIdxs {
		tp := refType[bi]
		if tp == "" {
			tp = "unknown"
		}
		cnt := refCount[bi]

		item := BlockItem{
			Index:    bi,
			Type:     tp,
			RefCount: cnt,
		}

		raw, err := readBlockBytes(mp, sb, bi)
		if err != nil {
			rep.Blocks = append(rep.Blocks, item)
			continue
		}

		switch tp {
		case "dir":
			item.Dir = &DirBlockView{Entries: parseDirEntries(raw)}
		case "file":
			size, prev := parseFilePreview(raw)
			item.File = &FileBlockView{Size: size, Preview: prev}
		case "ptr":
			item.Ptr = &PtrBlockView{Pointers: parsePointers(raw)}
		default:
			if esDirHeur(raw) {
				item.Type = "dir"
				item.Dir = &DirBlockView{Entries: parseDirEntries(raw)}
			} else {
				size, prev := parseFilePreview(raw)
				if size > 0 {
					item.Type = "file"
					item.File = &FileBlockView{Size: size, Preview: prev}
				} else {
					ps := parsePointers(raw)
					valid := 0
					for _, v := range ps {
						if v >= 0 {
							valid++
						}
					}
					if valid > 0 {
						item.Type = "ptr"
						item.Ptr = &PtrBlockView{Pointers: ps}
					}
				}
			}
		}

		rep.Blocks = append(rep.Blocks, item)
	}

	return rep, nil
}
