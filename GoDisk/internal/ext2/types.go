package ext2

// Constantes del FS
const (
	BlockSize          = 64
	FileSystemType     = 2
	FileSystemTypeEXT3 = 3
	MagicEXT2          = 0xEF53
	InodeDirectCount   = 15
)

// SuperBloque
type SuperBloque struct {
	SFilesystemType  int32
	SInodesCount     int32
	SBlocksCount     int32
	SFreeBlocksCount int32
	SFreeInodesCount int32
	SMtime           int64
	SUmtime          int64
	SMntCount        int32
	SMagic           int32
	SInodeS          int32
	SBlockS          int32
	SFirtsIno        int32
	SFirstBlo        int32
	SBmInodeStart    int64
	SBmBlockStart    int64
	SInodeStart      int64
	SBlockStart      int64
}

type Inodo struct {
	IUid   int32
	IGid   int32
	ISize  int32
	IAtime int64
	ICtime int64
	IMtime int64
	IBlock [InodeDirectCount]int32
	IType  byte
	IPerm  [3]byte
}

type DirEntry struct {
	BName  [12]byte
	BInodo int32
}
type BlockFolder struct {
	BContent [4]DirEntry
}
type BlockFile struct {
	BContent [BlockSize]byte
}

type BlockPointers struct {
	BPointers [16]int32
}
