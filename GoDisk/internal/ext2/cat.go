package ext2

import (
	"errors"
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func ReadFileByPath(reg *mount.Registry, id, absPath string) (Inodo, []byte, error) {
	mp, ok := reg.GetByID(id)
	if !ok {
		return Inodo{}, nil, fmt.Errorf("cat: id %s no está montado", id)
	}

	// SB
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return Inodo{}, nil, fmt.Errorf("cat: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return Inodo{}, nil, errors.New("cat: la partición no es EXT2 válida (SB)")
	}

	// Ruta -> componentes
	comps, err := splitPath(absPath)
	if err != nil {
		return Inodo{}, nil, err
	}
	if len(comps) == 0 {
		return Inodo{}, nil, errors.New("cat: path apunta a '/' (no es archivo)")
	}

	cur := int32(0)
	if len(comps) > 1 {
		for i := 0; i < len(comps)-1; i++ {
			next := lookupInDir(mp, sb, cur, comps[i])
			if next < 0 {
				return Inodo{}, nil, fmt.Errorf("cat: carpeta faltante '/%s'", comps[i])
			}
			ino, err := readInodeAt(mp, sb, next)
			if err != nil {
				return Inodo{}, nil, err
			}
			if ino.IType != 0 {
				return Inodo{}, nil, fmt.Errorf("cat: '/%s' no es carpeta", comps[i])
			}
			cur = next
		}
	}

	// Buscar archivo final
	name := comps[len(comps)-1]
	target := lookupInDir(mp, sb, cur, name)
	if target < 0 {
		return Inodo{}, nil, fmt.Errorf("cat: '%s' no existe", absPath)
	}
	ino, err := readInodeAt(mp, sb, target)
	if err != nil {
		return Inodo{}, nil, err
	}
	if ino.IType != 1 {
		return Inodo{}, nil, fmt.Errorf("cat: '%s' no es un archivo", absPath)
	}

	// Leer contenido según i_size usando punteros directos
	total := int(ino.ISize)
	if total < 0 {
		total = 0
	}
	var out []byte
	for _, p := range ino.IBlock {
		if p < 0 {
			continue
		}
		b, err := readFileBlockAt(mp, sb, p)
		if err != nil {
			return Inodo{}, nil, err
		}
		out = append(out, b.BContent[:]...)
		if len(out) >= total {
			break
		}
	}
	if len(out) > total {
		out = out[:total]
	}
	return ino, out, nil
}
