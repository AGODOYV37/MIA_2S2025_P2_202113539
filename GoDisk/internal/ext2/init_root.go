package ext2

import "time"

const usersBootstrap = "1, G, root \n1, U, root, root, 123 \n"

func newInodoCarpeta() Inodo {
	now := time.Now().Unix()
	ino := Inodo{
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

func newInodoArchivo(size int) Inodo {
	now := time.Now().Unix()
	ino := Inodo{
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

func buildRootBlock() BlockFolder {
	var bf BlockFolder
	copy(bf.BContent[0].BName[:], []byte("."))
	bf.BContent[0].BInodo = 0
	copy(bf.BContent[1].BName[:], []byte(".."))
	bf.BContent[1].BInodo = 0
	copy(bf.BContent[2].BName[:], []byte("users.txt"))
	bf.BContent[2].BInodo = 1

	return bf
}

func buildUsersBlock() BlockFile {
	var b BlockFile
	copy(b.BContent[:], []byte(usersBootstrap))
	return b
}
