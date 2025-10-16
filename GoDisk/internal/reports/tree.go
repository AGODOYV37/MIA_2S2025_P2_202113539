package reports

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// ===================== Modelo JSON =====================

type TreeReport struct {
	Kind       string      `json:"kind"` // "tree"
	DiskPath   string      `json:"diskPath"`
	ID         string      `json:"id"`
	BlockSize  int32       `json:"blockSize"`
	Inodes     int32       `json:"inodes"`
	Blocks     int32       `json:"blocks"`
	UsedInodes int         `json:"usedInodes"`
	UsedBlocks int         `json:"usedBlocks"`
	BlocksUsed []int32     `json:"blocksUsed"`
	Nodes      []TreeInode `json:"nodes"`
	Edges      []TreeEdge  `json:"edges"`
}

type BlocksExpanded struct {
	Direct         []int32            `json:"direct"`
	Indirect       *IndirectExpanded  `json:"indirect,omitempty"`
	DoubleIndirect *DoubleIndirectExp `json:"doubleIndirect,omitempty"`
}
type IndirectExpanded struct {
	Block    int32   `json:"block"`
	Pointers []int32 `json:"pointers"`
}
type PtrGroup struct {
	Block    int32   `json:"block"`
	Pointers []int32 `json:"pointers"`
}
type DoubleIndirectExp struct {
	Block  int32      `json:"block"`
	Groups []PtrGroup `json:"groups"`
}

type TreeInode struct {
	Index       int32           `json:"index"`
	Type        string          `json:"type"`
	RawType     byte            `json:"rawType"`
	Size        int32           `json:"size"`
	UID         int32           `json:"uid"`
	GID         int32           `json:"gid"`
	Perm        string          `json:"perm"`
	Blocks      BlocksExpanded  `json:"blocks"`
	BlocksFlat  []int32         `json:"blocksFlat"`
	DirectCards []TreeBlockCard `json:"directCards"`
}

type TreeEdge struct {
	Parent int32  `json:"parent"`
	Name   string `json:"name"`
	Child  int32  `json:"child"`
}

type TreeBlockCard struct {
	Index int32         `json:"index"`
	Type  string        `json:"type"` // "dir" | "file"
	Dir   *TreeDirData  `json:"dir,omitempty"`
	File  *TreeFileData `json:"file,omitempty"`
}

type TreeDirEntry struct {
	Name  string `json:"name"`
	Inode int32  `json:"inode"`
}
type TreeDirData struct {
	Entries []TreeDirEntry `json:"entries"`
}
type TreeFileData struct {
	Size    int32  `json:"size"`
	Preview string `json:"preview,omitempty"`
}

// ===================== Build / Generate =====================

func BuildTree(reg *mount.Registry, id string) (TreeReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TreeReport{}, fmt.Errorf("rep tree: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return TreeReport{}, fmt.Errorf("rep tree: id %q no está montado", id)
	}

	sb, err := readSuperBlock(mp)
	if err != nil {
		return TreeReport{}, fmt.Errorf("rep tree: leyendo super bloque: %w", err)
	}

	bmIn, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return TreeReport{}, fmt.Errorf("rep tree: cargando bitmaps: %w", err)
	}

	var blocksUsed []int32
	for i := int32(0); i < sb.SBlocksCount; i++ {
		if isLikelyUsedBlock(mp, sb, i, bmBl) {
			blocksUsed = append(blocksUsed, i)
		}
	}
	sort.Slice(blocksUsed, func(i, j int) bool { return blocksUsed[i] < blocksUsed[j] })

	nodes := make(map[int32]TreeInode)
	var edges []TreeEdge
	usedInodes := 0

	for idx := int32(0); idx < sb.SInodesCount; idx++ {
		if bmIn[idx] == 0 {
			continue
		}
		usedInodes++

		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			continue
		}

		flat := flattenBlocks(mp, sb, ino, bmBl)
		exp := buildBlocksExpanded(mp, sb, ino, bmBl)
		cards := buildDirectBlockCards(mp, sb, ino, bmBl)

		node := TreeInode{
			Index:       idx,
			Type:        decodeType(ino.IType),
			RawType:     ino.IType,
			Size:        ino.ISize,
			UID:         ino.IUid,
			GID:         ino.IGid,
			Perm:        decodePerm(ino.IPerm[:]),
			Blocks:      exp,
			BlocksFlat:  flat,
			DirectCards: cards,
		}
		nodes[idx] = node

		if node.Type == "dir" {
			children := readDirChildrenAllPointers(mp, sb, ino, bmBl)
			for _, ch := range children {
				edges = append(edges, TreeEdge{Parent: idx, Name: ch.Name, Child: ch.Inode})
			}
		}
	}

	idxs := make([]int32, 0, len(nodes))
	for k := range nodes {
		idxs = append(idxs, k)
	}
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] < idxs[j] })

	var outNodes []TreeInode
	for _, i := range idxs {
		outNodes = append(outNodes, nodes[i])
	}

	rep := TreeReport{
		Kind:       "tree",
		DiskPath:   mp.DiskPath,
		ID:         id,
		BlockSize:  ext2.BlockSize,
		Inodes:     sb.SInodesCount,
		Blocks:     sb.SBlocksCount,
		UsedInodes: usedInodes,
		UsedBlocks: len(blocksUsed),
		BlocksUsed: blocksUsed,
		Nodes:      outNodes,
		Edges:      edges,
	}
	return rep, nil
}

func GenerateTree(reg *mount.Registry, id, outPath string) error {
	rep, err := BuildTree(reg, id)
	if err != nil {
		return err
	}
	finalPath, format := resolveOutPathTree(outPath, id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("rep tree: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(finalPath, rep)
	case "html":
		html := RenderHTMLTree(rep)
		return os.WriteFile(finalPath, []byte(html), 0o644)
	default:
		return fmt.Errorf("rep tree: formato no soportado")
	}
}

// ===================== Helpers de lectura =====================

type dirChild struct {
	Name  string
	Inode int32
}

func readDirChildrenAllPointers(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo, bmBl []byte) []dirChild {
	var out []dirChild

	readDirBlock := func(b int32) {
		if !isLikelyUsedBlock(mp, sb, b, bmBl) {
			return
		}
		raw, err := readBlockBytes(mp, sb, b)
		if err != nil {
			return
		}
		ents := parseDirEntries(raw)
		for _, e := range ents {
			if e.Name == "." || e.Name == ".." || e.Inode < 0 || e.Inode >= sb.SInodesCount {
				continue
			}
			out = append(out, dirChild{Name: e.Name, Inode: e.Inode})
		}
	}

	for i, b := range ino.IBlock {
		if b < 0 || b >= sb.SBlocksCount {
			continue
		}
		switch {
		case i < 12:
			readDirBlock(b)
		case i == 12:
			for _, db := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				readDirBlock(db)
			}
		case i == 13:
			for _, mid := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				for _, db := range readPtrBlockTolerant(mp, sb, mid, bmBl) {
					readDirBlock(db)
				}
			}
		}
	}
	return out
}

func buildBlocksExpanded(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo, bmBl []byte) BlocksExpanded {
	var exp BlocksExpanded
	exp.Direct = make([]int32, 0, 12)

	for i, b := range ino.IBlock {
		if b < 0 || b >= sb.SBlocksCount {
			continue
		}
		switch {
		case i < 12:
			if isLikelyUsedBlock(mp, sb, b, bmBl) {
				exp.Direct = append(exp.Direct, b)
			}

		case i == 12:
			if isLikelyUsedBlock(mp, sb, b, bmBl) {
				ptrs := readPtrBlockTolerant(mp, sb, b, bmBl)
				exp.Indirect = &IndirectExpanded{Block: b, Pointers: ptrs}
			}

		case i == 13:
			if isLikelyUsedBlock(mp, sb, b, bmBl) {
				var groups []PtrGroup
				for _, mid := range readPtrBlockTolerant(mp, sb, b, bmBl) {
					ptrs := readPtrBlockTolerant(mp, sb, mid, bmBl)
					groups = append(groups, PtrGroup{Block: mid, Pointers: ptrs})
				}
				exp.DoubleIndirect = &DoubleIndirectExp{Block: b, Groups: groups}
			}
		}
	}
	return exp
}

func buildDirectBlockCards(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo, bmBl []byte) []TreeBlockCard {
	var out []TreeBlockCard

	isDir := strings.HasPrefix(decodeType(ino.IType), "dir")
	isFile := strings.HasPrefix(decodeType(ino.IType), "file")

	for i := 0; i < 12 && i < len(ino.IBlock); i++ {
		b := ino.IBlock[i]
		if !isLikelyUsedBlock(mp, sb, b, bmBl) {
			continue
		}
		raw, err := readBlockBytes(mp, sb, b)
		if err != nil {
			continue
		}

		if isDir {
			ents := parseDirEntries(raw)
			card := TreeBlockCard{
				Index: b,
				Type:  "dir",
				Dir:   &TreeDirData{Entries: make([]TreeDirEntry, 0, len(ents))},
			}
			for _, e := range ents {
				if e.Name == "." || e.Name == ".." || e.Inode < 0 || e.Inode >= sb.SInodesCount {
					continue
				}
				card.Dir.Entries = append(card.Dir.Entries, TreeDirEntry{
					Name:  e.Name,
					Inode: e.Inode,
				})
			}
			out = append(out, card)
			continue
		}

		if isFile {
			out = append(out, TreeBlockCard{
				Index: b,
				Type:  "file",
				File: &TreeFileData{
					Size:    ino.ISize,
					Preview: makePreview(raw, 256),
				},
			})
			continue
		}

		out = append(out, TreeBlockCard{
			Index: b,
			Type:  "file",
			File: &TreeFileData{
				Size:    ino.ISize,
				Preview: makePreview(raw, 256),
			},
		})
	}
	return out
}

func makePreview(b []byte, maxLen int) string {
	if len(b) > maxLen {
		b = b[:maxLen]
	}

	runes := make([]rune, 0, len(b))
	for _, by := range b {
		switch {
		case by == '\n' || by == '\r' || by == '\t' || (by >= 32 && by <= 126):
			runes = append(runes, rune(by))
		default:
			runes = append(runes, '.')
		}
	}

	s := strings.TrimRight(string(runes), "\x00 \r\n\t")
	return s
}

func flattenBlocks(mp *mount.MountedPartition, sb ext2.SuperBloque, ino ext2.Inodo, bmBl []byte) []int32 {
	var out []int32
	add := func(b int32) {
		if isLikelyUsedBlock(mp, sb, b, bmBl) {
			out = append(out, b)
		}
	}
	for i, b := range ino.IBlock {
		if b < 0 || b >= sb.SBlocksCount {
			continue
		}
		switch {
		case i < 12:
			add(b)
		case i == 12:
			add(b)
			for _, db := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				add(db)
			}
		case i == 13:
			add(b)
			for _, mid := range readPtrBlockTolerant(mp, sb, b, bmBl) {
				add(mid)
				for _, db := range readPtrBlockTolerant(mp, sb, mid, bmBl) {
					add(db)
				}
			}
		}
	}
	if len(out) == 0 {
		return out
	}
	set := make(map[int32]struct{}, len(out))
	for _, v := range out {
		set[v] = struct{}{}
	}
	flat := make([]int32, 0, len(set))
	for v := range set {
		flat = append(flat, v)
	}
	sort.Slice(flat, func(i, j int) bool { return flat[i] < flat[j] })
	return flat
}

// === Punteros tolerantes ===

func isLikelyUsedBlock(mp *mount.MountedPartition, sb ext2.SuperBloque, b int32, bmBl []byte) bool {
	if b < 0 || b >= sb.SBlocksCount {
		return false
	}
	if len(bmBl) > int(b) && bmBl[b] != 0 {
		return true
	}
	raw, err := readBlockBytes(mp, sb, b)
	if err != nil {
		return false
	}
	for _, by := range raw {
		if by != 0 {
			return true
		}
	}
	return false
}

func readPtrBlockTolerant(mp *mount.MountedPartition, sb ext2.SuperBloque, blk int32, bmBl []byte) []int32 {
	if !isLikelyUsedBlock(mp, sb, blk, bmBl) {
		return nil
	}
	raw, err := readBlockBytes(mp, sb, blk)
	if err != nil {
		return nil
	}
	out := make([]int32, 0, ext2.BlockSize/4)
	for i := 0; i+4 <= len(raw); i += 4 {
		v := int32(binary.LittleEndian.Uint32(raw[i : i+4]))
		if v >= 0 && v < sb.SBlocksCount && isLikelyUsedBlock(mp, sb, v, bmBl) {
			out = append(out, v)
		}
	}
	return out
}

// ===================== Salidas =====================

func resolveOutPathTree(out, id string) (string, string) {
	out = strings.TrimSpace(out)
	base := fmt.Sprintf("tree_%s.json", id)
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

func RenderHTMLTree(rep TreeReport) string {
	idx := make(map[int32]TreeInode)
	for _, n := range rep.Nodes {
		idx[n.Index] = n
	}
	children := make(map[int32][]TreeEdge)
	for _, e := range rep.Edges {
		children[e.Parent] = append(children[e.Parent], e)
	}
	for k := range children {
		sort.Slice(children[k], func(i, j int) bool {
			return strings.Compare(children[k][i].Name, children[k][j].Name) < 0
		})
	}

	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>TREE Report</title>")
	b.WriteString(`<style>
body{font-family:system-ui,Segoe UI,Roboto,Arial;margin:16px}
small{color:#666}
ul{list-style:disc}
code{background:#f7f7f7;padding:.15rem .3rem;border-radius:4px}
.block{margin-left:1rem}
table{border-collapse:collapse;margin-top:8px}
td,th{border:1px solid #ccc;padding:.35rem .5rem}
th{background:#f5f5f5}
</style>`)
	fmt.Fprintf(&b, "<h2>TREE (full)</h2><p><small>Disco: %s &nbsp; ID: %s &nbsp; Inodos usados: %d/%d &nbsp; Bloques usados: %d/%d</small></p>",
		escape(rep.DiskPath), escape(rep.ID), rep.UsedInodes, rep.Inodes, rep.UsedBlocks, rep.Blocks)

	b.WriteString("<h3>Inodos</h3><table><thead><tr><th>#</th><th>Tipo</th><th>Tamaño</th><th>UID/GID</th><th>Perm</th><th>Bloques</th></tr></thead><tbody>")
	for _, n := range rep.Nodes {
		blocks := make([]string, 0, len(n.BlocksFlat))
		for _, v := range n.BlocksFlat {
			blocks = append(blocks, fmt.Sprintf("%d", v))
		}
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%d</td><td>%d/%d</td><td>%s</td><td>%s</td></tr>",
			n.Index, escape(n.Type), n.Size, n.UID, n.GID, escape(n.Perm), escape(strings.Join(blocks, ", ")))
	}
	b.WriteString("</tbody></table>")

	b.WriteString("<h3>Relaciones directorios → hijos</h3><ul>")
	sort.Slice(rep.Edges, func(i, j int) bool {
		if rep.Edges[i].Parent == rep.Edges[j].Parent {
			return strings.Compare(rep.Edges[i].Name, rep.Edges[j].Name) < 0
		}
		return rep.Edges[i].Parent < rep.Edges[j].Parent
	})
	var last int32 = -1
	for _, e := range rep.Edges {
		if e.Parent != last {
			if last != -1 {
				b.WriteString("</ul></li>")
			}
			last = e.Parent
			fmt.Fprintf(&b, "<li><b>inode %d</b><ul>", e.Parent)
		}
		fmt.Fprintf(&b, "<li>%s → inode %d</li>", escape(e.Name), e.Child)
	}
	if last != -1 {
		b.WriteString("</ul></li>")
	}
	b.WriteString("</ul>")

	b.WriteString("<h3>Bloques usados</h3><p>")
	vals := make([]string, 0, len(rep.BlocksUsed))
	for _, v := range rep.BlocksUsed {
		vals = append(vals, fmt.Sprintf("%d", v))
	}
	b.WriteString(escape(strings.Join(vals, ", ")))
	b.WriteString("</p>")

	return b.String()
}
