package ext2

import (
	"fmt"
	"unicode/utf8"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func removeDirEntry(mp *mount.MountedPartition, sb SuperBloque, dirIno int32, name string) error {
	if name == "" || !utf8.ValidString(name) {
		return fmt.Errorf("removeDirEntry: nombre inválido")
	}
	ino, err := readInodeAt(mp, sb, dirIno)
	if err != nil {
		return err
	}
	for _, ptr := range ino.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if trimNull(bf.BContent[i].BName[:]) == name && bf.BContent[i].BInodo >= 0 {
				// limpiar la entrada
				for j := range bf.BContent[i].BName {
					bf.BContent[i].BName[j] = 0
				}
				bf.BContent[i].BInodo = -1
				return writeFolderBlockAt(mp, sb, ptr, bf)
			}
		}
	}
	return fmt.Errorf("removeDirEntry: '%s' no encontrado en el directorio", name)
}
