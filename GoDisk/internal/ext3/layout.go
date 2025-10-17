package ext3

import (
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/xbin"
)

const (
	FileSystemTypeExt3       = 3
	JournalEntrySize   int64 = 50 // enunciado
)

// ComputeLayoutExt3 devuelve: n, SB (con offsets), offset y tamaño del área de journal.
func ComputeLayoutExt3(partSize int64) (int32, ext2.SuperBloque, int64, int64, error) {
	var sb ext2.SuperBloque

	szSB := xbin.SizeOf[ext2.SuperBloque]()
	szIn := xbin.SizeOf[ext2.Inodo]()
	szBlk := int64(ext2.BlockSize)

	// Fórmula del enunciado:
	// size = sizeof(SB) + n*sizeOf(Journaling) + n + 3n + n*sizeOf(inodos) + 3n*sizeOf(block)
	den := JournalEntrySize + 1 + 3 + szIn + 3*szBlk
	n64 := (partSize - szSB) / den
	if n64 < 2 {
		return 0, sb, 0, 0, ext2.ErrPartTooSmall
	}
	n := int32(n64)

	// Offsets
	sbOff := int64(0)
	journalOff := sbOff + szSB                        // inmediatamente tras el SB
	bmInOff := journalOff + int64(n)*JournalEntrySize // journal ocupa n entradas
	bmBlOff := bmInOff + int64(n)                     // 1 byte por inodo
	inTblOff := bmBlOff + 3*int64(n)                  // 1 byte por bloque * 3n
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
		SMagic:           ext2.MagicEXT2, // ext3 usa el mismo magic 0xEF53
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
