package ext3

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/xbin"
)

// -------------------- Reporte --------------------

type ReplayReport struct {
	Total   int            `json:"total"`
	Applied int            `json:"applied"`
	Skipped int            `json:"skipped"`
	Failed  int            `json:"failed"`
	ByOp    map[string]int `json:"by_op"`
	Details []string       `json:"details"`
}

// -------------------- util local --------------------

func trimNull(b []byte) string {
	i := len(b)
	for i > 0 && b[i-1] == 0 {
		i--
	}
	return strings.TrimSpace(string(b[:i]))
}

func parseKV(s string) map[string]string {
	m := map[string]string{}
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	for _, f := range fields {
		if kv := strings.SplitN(f, "=", 2); len(kv) == 2 {
			k := strings.ToLower(strings.TrimSpace(kv[0]))
			v := strings.TrimSpace(kv[1])
			m[k] = v
		}
	}
	return m
}
func pbool(m map[string]string, k string, def bool) bool {
	v, ok := m[k]
	if !ok {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "t", "true", "yes", "y":
		return true
	case "0", "f", "false", "no", "n":
		return false
	default:
		return def
	}
}
func pint(m map[string]string, k string, def int) int {
	v, ok := m[k]
	if !ok {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}
func genData(n int) []byte {
	if n <= 0 {
		return nil
	}
	const pat = "0123456789"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = pat[i%len(pat)]
	}
	return b
}

// Usa readAt() y writeAt() ya existentes en este paquete (journal.go, io.go)

func readAllJournalEntries(mp *mount.MountedPartition, sb ext2.SuperBloque) ([]structs.Journal, error) {
	jOff, cap := journalRegion(sb) // journalRegion ya existe en ext3/journal.go
	if cap <= 0 {
		return nil, nil
	}
	entrySize := int64(xbin.SizeOf[structs.Journal]())

	var cur structs.Journal
	var out []structs.Journal
	for i := int64(0); i < cap; i++ {
		off := mp.Start + jOff + i*entrySize
		if err := readAt(mp.DiskPath, off, &cur); err != nil {
			return nil, fmt.Errorf("recovery: leyendo journal[%d]: %w", i, err)
		}
		if cur.JCount > 0 {
			out = append(out, cur)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JCount < out[j].JCount })
	return out, nil
}

// -------------------- Recovery con Reporte --------------------

// RecoverWithReport reconstruye el FS EXT3 con journal y retorna un reporte detallado.
// Regresa error solo en fallas críticas (leer SB, leer journal, mkfs). Las fallas por
// entrada se registran en el reporte y la ejecución continúa.
func RecoverWithReport(reg *mount.Registry, id string) (ReplayReport, error) {
	rep := ReplayReport{
		ByOp: make(map[string]int),
	}

	mp, ok := reg.GetByID(id)
	if !ok {
		return rep, fmt.Errorf("recovery: id %s no está montado", id)
	}
	var sb ext2.SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return rep, fmt.Errorf("recovery: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemTypeExt3 {
		return rep, errors.New("recovery: solo aplica para particiones EXT3")
	}

	entries, err := readAllJournalEntries(mp, sb)
	if err != nil {
		return rep, err
	}

	// 1) Re-formatear EXT3
	if err := NewFormatter(reg).MkfsFull(id); err != nil {
		return rep, fmt.Errorf("recovery: mkfs ext3: %w", err)
	}

	// 2) Re-aplicar como root
	const rootUID, rootGID = 1, 1

	for _, e := range entries {
		rep.Total++

		// Nota: ajusta los nombres de campos según tu structs.JournalInfo:
		// Si los tienes en camelCase (IOperation, IPath, IContent), usa esos.
		op := strings.ToUpper(strings.TrimSpace(trimNull(e.JContent.I_operation[:])))
		pth := strings.TrimSpace(trimNull(e.JContent.I_path[:]))
		raw := strings.TrimSpace(trimNull(e.JContent.I_content[:]))
		kv := parseKV(raw)

		applyOK := func() {
			rep.Applied++
			rep.ByOp[op]++
		}
		fail := func(format string, a ...any) {
			rep.Failed++
			rep.Details = append(rep.Details, fmt.Sprintf(format, a...))
		}
		skip := func(format string, a ...any) {
			rep.Skipped++
			rep.Details = append(rep.Details, fmt.Sprintf(format, a...))
		}

		switch op {
		case "MKDIR":
			if err := ext2.MakeDir(reg, id, pth, true, rootUID, rootGID); err != nil {
				fail("MKDIR %q: %v", pth, err)
				continue
			}
			applyOK()

		case "MKFILE":
			data := []byte(raw)
			if len(data) == 0 {
				size := pint(kv, "size", 0)
				data = genData(size)
			}
			if err := ext2.CreateOrOverwriteFile(reg, id, pth, data, true, true, rootUID, rootGID); err != nil {
				fail("MKFILE %q: %v", pth, err)
				continue
			}
			applyOK()

		case "EDIT":
			data := []byte(raw)
			if err := ext2.EditFile(reg, id, pth, data, rootUID, rootGID, true); err != nil {
				fail("EDIT %q: %v", pth, err)
				continue
			}
			applyOK()

		case "COPY":
			dst := kv["dest"]
			if dst == "" {
				skip("COPY %q: falta dest=", pth)
				continue
			}
			if err := ext2.CopyNode(reg, id, pth, dst, rootUID, rootGID, true); err != nil {
				fail("COPY %q->%q: %v", pth, dst, err)
				continue
			}
			applyOK()

		case "MOVE":
			dst := kv["dest"]
			if dst == "" {
				skip("MOVE %q: falta dest=", pth)
				continue
			}
			if err := ext2.MoveNode(reg, id, pth, dst, rootUID, rootGID, true); err != nil {
				fail("MOVE %q->%q: %v", pth, dst, err)
				continue
			}
			applyOK()

		case "REMOVE":
			if err := ext2.Remove(reg, id, pth, rootUID, rootGID); err != nil {
				fail("REMOVE %q: %v", pth, err)
				continue
			}
			applyOK()

		case "RENAME":
			newName := kv["name"]
			if newName == "" {
				skip("RENAME %q: falta name=", pth)
				continue
			}
			if err := ext2.RenameNode(reg, id, pth, newName, rootUID, rootGID, true); err != nil {
				fail("RENAME %q->%q: %v", pth, newName, err)
				continue
			}
			applyOK()

		case "CHMOD":
			ugo := kv["ugo"]
			perms, perr := ext2.ParseUGO(ugo)
			if perr != nil {
				fail("CHMOD %q: %v", pth, perr)
				continue
			}
			rec := pbool(kv, "r", false)
			if err := ext2.Chmod(reg, id, pth, perms, rec, rootUID, rootGID, true); err != nil {
				fail("CHMOD %q: %v", pth, err)
				continue
			}
			applyOK()

		case "CHOWN":
			user := kv["usuario"]
			if user == "" {
				skip("CHOWN %q: falta usuario=", pth)
				continue
			}
			rec := pbool(kv, "r", false)
			if err := ext2.Chown(reg, id, pth, user, rec, rootUID, rootGID, true); err != nil {
				fail("CHOWN %q: %v", pth, err)
				continue
			}
			applyOK()

		default:
			skip("op desconocida %q (path=%q)", op, pth)
		}
	}
	return rep, nil
}

// Mantén Recover para compatibilidad; retorna error solo si falla algo crítico.
func Recover(reg *mount.Registry, id string) error {
	_, err := RecoverWithReport(reg, id)
	return err
}
