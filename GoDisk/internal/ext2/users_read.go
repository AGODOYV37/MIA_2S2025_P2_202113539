package ext2

import (
	"errors"
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func ReadUsersText(reg *mount.Registry, id string) (string, error) {
	mp, ok := reg.GetByID(id)
	if !ok {
		return "", fmt.Errorf("users: id %s no está montado", id)
	}

	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return "", fmt.Errorf("users: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return "", errors.New("users: la partición no parece EXT2 válida")
	}

	root, err := readInodeAt(mp, sb, 0)
	if err != nil {
		return "", fmt.Errorf("users: leyendo inodo raíz: %w", err)
	}
	usersIno := int32(-1)
	for _, ptr := range root.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return "", err
		}
		for _, de := range bf.BContent {
			name := trimNull(de.BName[:])
			if name == "users.txt" && de.BInodo >= 0 {
				usersIno = de.BInodo
				break
			}
		}
		if usersIno >= 0 {
			break
		}
	}

	if usersIno < 0 {
		usersIno = 1
	}

	uino, err := readInodeAt(mp, sb, usersIno)
	if err != nil {
		return "", fmt.Errorf("users: leyendo inodo users.txt: %w", err)
	}
	size := int(uino.ISize)
	if size < 0 {
		size = 0
	}

	var out []byte
	for _, ptr := range uino.IBlock {
		if ptr < 0 {
			continue
		}
		b, err := readFileBlockAt(mp, sb, ptr)
		if err != nil {
			return "", err
		}
		out = append(out, b.BContent[:]...)
		if len(out) >= size {
			break
		}
	}
	if len(out) > size {
		out = out[:size]
	}
	return string(out), nil
}
