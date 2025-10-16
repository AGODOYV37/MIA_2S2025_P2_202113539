package ext2

import (
	"encoding/binary"
	"time"
)

func sizeof[T any](v T) int64 { return int64(binary.Size(v)) }

func ComputeLayout(partSize int64) (int32, SuperBloque, error) {
	var sb SuperBloque
	var dummySB SuperBloque
	var dummyIn Inodo

	szSB := sizeof(dummySB)
	szIn := sizeof(dummyIn)
	szBlk := int64(BlockSize)

	den := int64(4) + szIn + 3*szBlk
	n64 := (partSize - szSB) / den
	if n64 < 2 {
		return 0, sb, ErrPartTooSmall
	}
	n := int32(n64)

	off := int64(0)
	sbOff := off
	bmInOff := sbOff + szSB
	bmBlOff := bmInOff + int64(n)
	inTblOff := bmBlOff + 3*int64(n)
	blkTblOff := inTblOff + int64(n)*szIn

	sb = SuperBloque{
		SFilesystemType:  FileSystemType,
		SInodesCount:     n,
		SBlocksCount:     3 * n,
		SFreeInodesCount: n,
		SFreeBlocksCount: 3 * n,
		SMtime:           time.Now().Unix(),
		SUmtime:          0,
		SMntCount:        1,
		SMagic:           MagicEXT2,
		SInodeS:          int32(szIn),
		SBlockS:          int32(szBlk),
		SFirtsIno:        0,
		SFirstBlo:        0,
		SBmInodeStart:    bmInOff,
		SBmBlockStart:    bmBlOff,
		SInodeStart:      inTblOff,
		SBlockStart:      blkTblOff,
	}
	return n, sb, nil
}
