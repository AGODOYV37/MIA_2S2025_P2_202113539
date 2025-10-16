package ext3

import (
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
)

const usersBootstrap = "1, G, root \n1, U, root, root, 123 \n"

func newInodoCarpeta() ext2.Inodo {
	now := time.Now().Unix()
	ino := ext2.Inodo{
		IUid: 1, IGid: 1, ISize: 0,
		IAtime: now, ICtime: now, IMtime: now,
		IType: 0,
		IPerm: [3]byte{7, 7, 5},
	}
	for i := range ino.IBlock {
		ino.IBlock[i] = -1
	}
	return ino
}

func newInodoArchivo(size int) ext2.Inodo {
	now := time.Now().Unix()
	ino := ext2.Inodo{
		IUid: 1, IGid: 1, ISize: int32(size),
		IAtime: now, ICtime: now, IMtime: now,
		IType: 1,
		IPerm: [3]byte{6, 6, 4},
	}
	for i := range ino.IBlock {
		ino.IBlock[i] = -1
	}
	return ino
}

func buildRootBlockExt3() ext2.BlockFolder {
	var bf ext2.BlockFolder
	copy(bf.BContent[0].BName[:], []byte("."))
	bf.BContent[0].BInodo = 0
	copy(bf.BContent[1].BName[:], []byte(".."))
	bf.BContent[1].BInodo = 0
	copy(bf.BContent[2].BName[:], []byte("users.txt"))
	bf.BContent[2].BInodo = 1
	copy(bf.BContent[3].BName[:], []byte(".journal"))
	bf.BContent[3].BInodo = 2
	return bf
}

func buildUsersBlock() ext2.BlockFile {
	var b ext2.BlockFile
	copy(b.BContent[:], []byte(usersBootstrap))
	return b
}

func buildJournalBlock() ext2.BlockFile {
	var b ext2.BlockFile
	// contenido vacío (cero) para simbolizar el journal
	return b
}
