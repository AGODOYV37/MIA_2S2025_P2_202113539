package ext2

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func ParseUGO(s string) ([3]byte, error) {
	var out [3]byte
	if len(s) != 3 {
		return out, errors.New("UGO debe tener 3 dígitos 0..7")
	}
	for i := 0; i < 3; i++ {
		r := s[i]
		if r < '0' || r > '7' {
			return out, fmt.Errorf("UGO fuera de rango en posición %d (0..7)", i+1)
		}
		out[i] = byte(r - '0')
	}
	return out, nil
}

// Chmod cambia los bits de permisos (U,G,O) en el nodo indicado.
func Chmod(reg *mount.Registry, id, absPath string, perms [3]byte, recursive bool, uid, gid int, isRoot bool) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("chmod: id %s no está montado", id)
	}
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("chmod: leyendo SB: %w", err)
	}
	if err := requireSupportedFS(sb, "chmod"); err != nil {
		return err
	}

	absPath = path.Clean(strings.TrimSpace(absPath))
	if !strings.HasPrefix(absPath, "/") {
		return errors.New("chmod: -path debe ser absoluto")
	}

	// Resolver inodo del path (soporta "/" – inodo 0)
	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	var idx int32
	if len(comps) == 0 {
		// raíz
		idx = 0
	} else {
		i, exists, err := resolvePathInode(mp, sb, comps)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("chmod: ruta no existe: %s", absPath)
		}
		idx = i
	}

	// función para aplicar permisos a un inodo
	setPerm := func(i int32) error {
		ino, err := readInodeAt(mp, sb, i)
		if err != nil {
			return err
		}
		ino.IPerm = perms
		return writeInodeAt(mp, sb, i, ino)
	}

	if !recursive {
		// No recursivo: cambia solo el nodo apuntado
		return setPerm(idx)
	}

	// Recursivo: recorrer todo el subárbol y aplicar SOLO a nodos del usuario actual.
	var walk func(i int32) error
	walk = func(i int32) error {
		ino, err := readInodeAt(mp, sb, i)
		if err != nil {
			return err
		}
		// Aplicar solo si el propietario es el usuario actual
		if int32(uid) == ino.IUid {
			if err := setPerm(i); err != nil {
				return err
			}
		}
		// Si es carpeta, bajar a los hijos
		if ino.IType == 0 {
			children, err := listDirEntries(mp, sb, i)
			if err != nil {
				return err
			}
			for _, ch := range children {
				if err := walk(ch.ino); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(idx)
}
