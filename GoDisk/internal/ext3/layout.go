package ext3

import (
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/xbin"
)

const (
	FileSystemTypeExt3       = 3
	JournalEntrySize   int64 = 50
)

func ComputeLayoutExt3(partSize int64) (int32, ext2.SuperBloque, int64, int64, error) {
	var sb ext2.SuperBloque

	szSB := xbin.SizeOf[ext2.SuperBloque]()
	szIn := xbin.SizeOf[ext2.Inodo]()
	szBlk := int64(ext2.BlockSize)

	den := JournalEntrySize + 1 + 3 + szIn + 3*szBlk
	n64 := (partSize - szSB) / den
	if n64 < 2 {
		return 0, sb, 0, 0, ext2.ErrPartTooSmall
	}
	n := int32(n64)

	sbOff := int64(0)
	journalOff := sbOff + szSB
	bmInOff := journalOff + int64(n)*JournalEntrySize
	bmBlOff := bmInOff + int64(n)
	inTblOff := bmBlOff + 3*int64(n)
	blkTblOff := inTblOff + int64(n)*szIn

	sb = ext2.SuperBloque{
		SFilesystemType:  FileSystemTypeExt3,
		SInodesCount:     n,
		SBlocksCount:     3 * n,
		SFreeInodesCount: n,
		SFreeBlocksCount: 3 * n,
		SMtime:           time.Now().Unix(),
		SUmtime:          0,
		SMntCount:        1,
		SMagic:           ext2.MagicEXT2,
		SInodeS:          int32(szIn),
		SBlockS:          int32(szBlk),
		SFirtsIno:        0,
		SFirstBlo:        0,
		SBmInodeStart:    bmInOff,
		SBmBlockStart:    bmBlOff,
		SInodeStart:      inTblOff,
		SBlockStart:      blkTblOff,
	}

	return n, sb, journalOff, int64(n) * JournalEntrySize, nil
}
