package reports

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

// ===================== Modelo =====================

type LSReport struct {
	Kind     string   `json:"kind"`
	DiskPath string   `json:"diskPath"`
	ID       string   `json:"id"`
	Dir      string   `json:"dir"`
	Items    []LSItem `json:"items"`
}

type LSItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	RawType byte   `json:"rawType"`

	Inode int32 `json:"inode"`
	Size  int32 `json:"size"`

	Perm  string `json:"perm"`
	UID   int32  `json:"uid"`
	GID   int32  `json:"gid"`
	Owner string `json:"owner"`
	Group string `json:"group"`

	MTime string `json:"mtime"`
	ATime string `json:"atime"`
	CTime string `json:"ctime"`
}

// ===================== Build / Generate =====================

func BuildLS(reg *mount.Registry, id, pathLS string) (LSReport, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return LSReport{}, fmt.Errorf("rep ls: -id requerido")
	}
	mp, ok := reg.GetByID(id)
	if !ok {
		return LSReport{}, fmt.Errorf("rep ls: id %q no está montado", id)
	}

	dirPath := strings.TrimSpace(pathLS)
	if dirPath == "" {
		dirPath = "/"
	}
	dirPath = normalizePath(dirPath)

	sb, err := readSuperBlock(mp)
	if err != nil {
		return LSReport{}, fmt.Errorf("rep ls: leyendo super bloque: %w", err)
	}

	bmIn, bmBl, err := loadBitmapsForReport(mp, sb)
	if err != nil {
		return LSReport{}, fmt.Errorf("rep ls: cargando bitmaps: %w", err)
	}

	idx, err := resolvePathToInode(mp, sb, bmBl, dirPath)
	if err != nil {
		return LSReport{}, fmt.Errorf("rep ls: %v", err)
	}
	if idx < 0 || idx >= sb.SInodesCount || bmIn[idx] == 0 {
		return LSReport{}, fmt.Errorf("rep ls: inodo fuera de rango o no usado")
	}
	ino, err := readInodeAt(mp, sb, idx)
	if err != nil {
		return LSReport{}, fmt.Errorf("rep ls: leyendo inodo #%d: %w", idx, err)
	}

	uidName, gidName := tryLoadUsersNames(mp, sb, bmBl)

	out := LSReport{
		Kind:     "ls",
		DiskPath: mp.DiskPath,
		ID:       id,
		Dir:      dirPath,
	}

	t := decodeType(ino.IType)
	if t == "dir" {

		children := readDirChildrenAllPointers(mp, sb, ino, bmBl)

		sort.Slice(children, func(i, j int) bool {
			return strings.Compare(children[i].Name, children[j].Name) < 0
		})

		for _, ch := range children {
			if ch.Name == "." || ch.Name == ".." {
				continue
			}
			if ch.Inode < 0 || ch.Inode >= sb.SInodesCount {
				continue
			}
			if bmIn[ch.Inode] == 0 {

				continue
			}
			cino, err := readInodeAt(mp, sb, ch.Inode)
			if err != nil {
				continue
			}
			item := inodeToLSItem(ch.Name, ch.Inode, cino, uidName, gidName)
			out.Items = append(out.Items, item)
		}

	} else {

		item := inodeToLSItem(filepath.Base(dirPath), idx, ino, uidName, gidName)
		out.Items = append(out.Items, item)
	}

	return out, nil
}

func GenerateLS(reg *mount.Registry, id, pathLS, outPath string) error {
	rep, err := BuildLS(reg, id, pathLS)
	if err != nil {
		return err
	}
	final, format := resolveOutPathLS(outPath, id, pathLS)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return fmt.Errorf("rep ls: creando carpeta destino: %w", err)
	}
	switch format {
	case "json":
		return writeJSON(final, rep)
	case "html":
		return writeHTML_LS(final, rep)
	default:
		return fmt.Errorf("rep ls: formato no soportado")
	}
}

// ===================== Helpers =====================

func inodeToLSItem(name string, idx int32, ino ext2.Inodo, uidName map[int32]string, gidName map[int32]string) LSItem {
	owner := uidName[ino.IUid]
	group := gidName[ino.IGid]
	return LSItem{
		Name:    name,
		Type:    decodeType(ino.IType),
		RawType: ino.IType,
		Inode:   idx,
		Size:    ino.ISize,
		Perm:    decodePerm(ino.IPerm[:]),
		UID:     ino.IUid,
		GID:     ino.IGid,
		Owner:   owner,
		Group:   group,
		MTime:   fmtTS(ino.IMtime),
		ATime:   fmtTS(ino.IAtime),
		CTime:   fmtTS(ino.ICtime),
	}
}

func fmtTS(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

func tryLoadUsersNames(mp *mount.MountedPartition, sb ext2.SuperBloque, bmBl []byte) (map[int32]string, map[int32]string) {
	uidToName := map[int32]string{}
	gidToName := map[int32]string{}

	usersIdx, err := resolvePathToInode(mp, sb, bmBl, "/users.txt")
	if err != nil || usersIdx < 0 || usersIdx >= sb.SInodesCount {
		return uidToName, gidToName
	}
	uIno, err := readInodeAt(mp, sb, usersIdx)
	if err != nil || decodeType(uIno.IType) != "file" {
		return uidToName, gidToName
	}
	raw, err := readWholeFile(mp, sb, uIno, bmBl)
	if err != nil {
		return uidToName, gidToName
	}

	if int64(len(raw)) > int64(uIno.ISize) && uIno.ISize >= 0 {
		raw = raw[:uIno.ISize]
	}
	txt := string(raw)

	for _, line := range splitLinesSafe(txt) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := splitCSVSafe(line)
		if len(parts) < 2 {
			continue
		}
		if strings.EqualFold(parts[1], "G") && len(parts) >= 3 {

			gid := intSafe(parts[0])
			name := strings.TrimSpace(parts[2])
			gidToName[int32(gid)] = name
		} else if strings.EqualFold(parts[1], "U") && len(parts) >= 4 {

			uid := intSafe(parts[0])
			username := strings.TrimSpace(parts[3])
			uidToName[int32(uid)] = username
		}
	}
	return uidToName, gidToName
}

func splitLinesSafe(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
func splitCSVSafe(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
func intSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// ===================== Salidas =====================

func resolveOutPathLS(out, id, lsPath string) (string, string) {
	out = strings.TrimSpace(out)
	baseName := "ls_" + id + ".json"
	if strings.TrimSpace(lsPath) != "" {

		clean := strings.Trim(strings.ReplaceAll(lsPath, "/", "_"), "_")
		if clean != "" {
			baseName = "ls_" + id + "_" + clean + ".json"
		}
	}
	if out == "" {
		return baseName, "json"
	}
	ext := strings.ToLower(filepath.Ext(out))
	if ext == ".json" {
		return out, "json"
	}
	if ext == ".html" || ext == ".htm" {
		return out, "html"
	}
	if st, err := os.Stat(out); err == nil && st.IsDir() {
		return filepath.Join(out, baseName), "json"
	}
	if ext == "" {
		return out + ".json", "json"
	}
	return out, "json"
}

func writeHTML_LS(path string, rep LSReport) error {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>LS Report</title>")
	b.WriteString(`<style>body{font-family:system-ui,Segoe UI,Roboto,Arial}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.4rem .6rem}th{background:#f5f5f5}</style>`)
	fmt.Fprintf(&b, "<h2>LS — %s</h2>", escape(rep.Dir))
	fmt.Fprintf(&b, "<p><b>Disk:</b> %s &nbsp;|&nbsp; <b>ID:</b> %s</p>", escape(rep.DiskPath), escape(rep.ID))

	b.WriteString("<table><thead><tr>")
	b.WriteString("<th>Name</th><th>Type</th><th>Inode</th><th>Size</th><th>Perm</th><th>UID</th><th>Owner</th><th>GID</th><th>Group</th><th>mtime</th><th>ctime</th><th>atime</th>")
	b.WriteString("</tr></thead><tbody>")
	for _, it := range rep.Items {
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			escape(it.Name), escape(it.Type), it.Inode, it.Size, escape(it.Perm),
			it.UID, escape(it.Owner), it.GID, escape(it.Group),
			escape(it.MTime), escape(it.CTime), escape(it.ATime))
	}
	b.WriteString("</tbody></table>")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func resolvePathToInode(mp *mount.MountedPartition, sb ext2.SuperBloque, bmBl []byte, path string) (int32, error) {
	p := normalizePath(path)
	if p == "" || p == "/" {
		return 0, nil // raíz
	}

	parts := strings.Split(strings.Trim(p, "/"), "/")
	cur := int32(0)

	for _, name := range parts {
		ino, err := readInodeAt(mp, sb, cur)
		if err != nil {
			return -1, fmt.Errorf("inode #%d: %w", cur, err)
		}

		children := readDirChildrenAllPointers(mp, sb, ino, bmBl)

		if decodeType(ino.IType) != "dir" && len(children) == 0 {
			return -1, fmt.Errorf("ruta intermedia no es carpeta: %q", name)
		}

		next := int32(-1)
		for _, ch := range children {
			if ch.Name == name {
				next = ch.Inode
				break
			}
		}
		if next < 0 {
			return -1, fmt.Errorf("no existe: %q", name)
		}
		cur = next
	}
	return cur, nil
}
