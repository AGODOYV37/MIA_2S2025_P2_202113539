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

// --- util local --- //
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

// Usa readAt() y writeAt() ya existentes en este paquete:
// - readAt: ext3/journal.go
// - writeAt: ext3/io.go

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

// Recover reconstruye el FS EXT3 usando el journal.
// Estrategia simple: 1) leer journal, 2) mkfs EXT3, 3) re-aplicar entradas como root.
func Recover(reg *mount.Registry, id string) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("recovery: id %s no está montado", id)
	}
	var sb ext2.SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("recovery: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemTypeExt3 {
		return errors.New("recovery: solo aplica para particiones EXT3")
	}

	entries, err := readAllJournalEntries(mp, sb)
	if err != nil {
		return err
	}

	// 1) Re-formatear EXT3
	if err := NewFormatter(reg).MkfsFull(id); err != nil {
		return fmt.Errorf("recovery: mkfs ext3: %w", err)
	}

	// 2) Re-aplicar como root
	const rootUID, rootGID = 1, 1

	for _, e := range entries {
		op := strings.ToUpper(strings.TrimSpace(trimNull(e.JContent.I_operation[:])))
		pth := strings.TrimSpace(trimNull(e.JContent.I_path[:]))
		inf := strings.TrimSpace(trimNull(e.JContent.I_content[:]))
		kv := parseKV(inf)

		switch op {
		case "MKDIR":
			if err := ext2.MakeDir(reg, id, pth, true, rootUID, rootGID); err != nil {
				return fmt.Errorf("recovery: MKDIR %q: %w", pth, err)
			}

		case "MKFILE":
			data := []byte(inf)
			if len(data) == 0 {
				size := pint(kv, "size", 0)
				data = genData(size)
			}
			if err := ext2.CreateOrOverwriteFile(reg, id, pth, data, true, true, rootUID, rootGID); err != nil {
				return fmt.Errorf("recovery: MKFILE %q: %w", pth, err)
			}

		case "EDIT":
			data := []byte(inf)
			if err := ext2.EditFile(reg, id, pth, data, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: EDIT %q: %w", pth, err)
			}

		case "COPY":
			dst := kv["dest"]
			if dst == "" {
				return fmt.Errorf("recovery: COPY %q: falta dest=", pth)
			}
			if err := ext2.CopyNode(reg, id, pth, dst, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: COPY %q->%q: %w", pth, dst, err)
			}

		case "MOVE":
			dst := kv["dest"]
			if dst == "" {
				return fmt.Errorf("recovery: MOVE %q: falta dest=", pth)
			}
			if err := ext2.MoveNode(reg, id, pth, dst, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: MOVE %q->%q: %w", pth, dst, err)
			}

		case "REMOVE":
			if err := ext2.Remove(reg, id, pth, rootUID, rootGID); err != nil {
				return fmt.Errorf("recovery: REMOVE %q: %w", pth, err)
			}

		case "RENAME":
			newName := kv["name"]
			if newName == "" {
				return fmt.Errorf("recovery: RENAME %q: falta name=", pth)
			}
			if err := ext2.RenameNode(reg, id, pth, newName, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: RENAME %q->%q: %w", pth, newName, err)
			}

		case "CHMOD":
			ugo := kv["ugo"]
			perms, err := ext2.ParseUGO(ugo)
			if err != nil {
				return fmt.Errorf("recovery: CHMOD %q: %v", pth, err)
			}
			rec := pbool(kv, "r", false)
			if err := ext2.Chmod(reg, id, pth, perms, rec, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: CHMOD %q: %w", pth, err)
			}

		case "CHOWN":
			user := kv["usuario"]
			if user == "" {
				return fmt.Errorf("recovery: CHOWN %q: falta usuario=", pth)
			}
			rec := pbool(kv, "r", false)
			if err := ext2.Chown(reg, id, pth, user, rec, rootUID, rootGID, true); err != nil {
				return fmt.Errorf("recovery: CHOWN %q: %w", pth, err)
			}

		default:
			// ignoramos operaciones no reconocidas
		}
	}
	return nil
}
