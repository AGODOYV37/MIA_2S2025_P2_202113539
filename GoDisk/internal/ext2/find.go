package ext2

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Find(reg *mount.Registry, id, startPath, pattern string, uid, gid int, isRoot bool) ([]string, error) {
	mp, ok := reg.GetByID(id)
	if !ok {
		return nil, fmt.Errorf("find: id %s no está montado", id)
	}
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return nil, fmt.Errorf("find: leyendo SB: %w", err)
	}
	if err := requireSupportedFS(sb, "find"); err != nil {
		return nil, err
	}

	startPath = path.Clean(strings.TrimSpace(startPath))
	if !strings.HasPrefix(startPath, "/") {
		return nil, errors.New("find: -path debe ser absoluto")
	}

	comps, err := splitPath(startPath)
	if err != nil {
		return nil, err
	}
	startIno, exists, err := resolvePathInode(mp, sb, comps)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("find: ruta no existe: %s", startPath)
	}

	startNode, err := readInodeAt(mp, sb, startIno)
	if err != nil {
		return nil, err
	}
	if startNode.IType != 0 {
		return nil, errors.New("find: -path debe ser una carpeta")
	}
	if !CanRead(startNode, uid, gid, isRoot) {
		return nil, fmt.Errorf("find: sin permiso de lectura en '%s'", startPath)
	}

	re, err := globToRegex(pattern)

	if err != nil {
		return nil, fmt.Errorf("find: patrón inválido: %v", err)
	}

	var out []string

	var walk func(idx int32, abs string) error
	walk = func(idx int32, abs string) error {
		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			return err
		}

		if ino.IType == 1 {
			base := path.Base(abs)
			if CanRead(ino, uid, gid, isRoot) && re.MatchString(base) {
				out = append(out, abs)
			}
			return nil
		}

		if !CanRead(ino, uid, gid, isRoot) {
			return nil
		}

		if abs != "/" && re.MatchString(path.Base(abs)) {
			out = append(out, abs)
		}

		entries, err := listDirEntries(mp, sb, idx)
		if err != nil {
			return err
		}
		for _, e := range entries {
			childAbs := path.Join(abs, e.name)
			if err := walk(e.ino, childAbs); err != nil {
				return err
			}
		}
		return nil
	}

	startAbs := startPath
	if startAbs == "" {
		startAbs = "/"
	}
	return out, walk(startIno, startAbs)
}

func globToRegex(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".+")
		case '?':
			b.WriteString(".")
		default:
			if strings.ContainsRune(`\.^$|()[]{}+*?`, r) {
				b.WriteRune('\\')
			}
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
