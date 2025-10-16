package ext3

import (
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

const FileSystemTypeExt3 = 3

type Formatter struct{ reg *mount.Registry }

func NewFormatter(reg *mount.Registry) *Formatter { return &Formatter{reg: reg} }

func (f *Formatter) MkfsFull(id string) error {
	mp, ok := f.reg.GetByID(id)
	if !ok {
		return fmt.Errorf("mkfs: id %s no está montado", id)
	}

	partStart := mp.Start
	partSize := mp.Size

	// Reutilizamos el layout de EXT2
	_, sb, err := ext2.ComputeLayout(partSize)
	if err != nil {
		return err
	}
	sb.SFilesystemType = FileSystemTypeExt3

	if err := writeAt(mp.DiskPath, partStart, sb); err != nil {
		return fmt.Errorf("mkfs: error escribiendo superbloque: %w", err)
	}

	bmIn, bmBl := ext2.NewBitmaps(sb.SInodesCount, sb.SBlocksCount)

	// Reservamos 3 inodos y 3 bloques: root, users.txt, .journal
	ext2.MarkInode(bmIn, 0, true)
	ext2.MarkBlock(bmBl, 0, true)
	ext2.MarkInode(bmIn, 1, true)
	ext2.MarkBlock(bmBl, 1, true)
	ext2.MarkInode(bmIn, 2, true)
	ext2.MarkBlock(bmBl, 2, true)

	if err := writeBytes(mp.DiskPath, partStart+sb.SBmInodeStart, bmIn); err != nil {
		return err
	}
	if err := writeBytes(mp.DiskPath, partStart+sb.SBmBlockStart, bmBl); err != nil {
		return err
	}

	// Inodos
	inoRoot := newInodoCarpeta()
	inoRoot.IBlock[0] = 0
	if err := writeAt(mp.DiskPath, partStart+sb.SInodeStart+0*int64(sb.SInodeS), inoRoot); err != nil {
		return err
	}

	users := buildUsersBlock()
	contentLen := len([]byte(usersBootstrap))
	inoUsers := newInodoArchivo(contentLen)
	inoUsers.IBlock[0] = 1
	if err := writeAt(mp.DiskPath, partStart+sb.SInodeStart+1*int64(sb.SInodeS), inoUsers); err != nil {
		return err
	}

	// Inodo de journal (simboliza el journal como archivo regular oculto)
	journal := buildJournalBlock()
	inoJournal := newInodoArchivo(len(journal.BContent))
	inoJournal.IBlock[0] = 2
	if err := writeAt(mp.DiskPath, partStart+sb.SInodeStart+2*int64(sb.SInodeS), inoJournal); err != nil {
		return err
	}

	// Bloques
	rootBlk := buildRootBlockExt3() // incluye ".journal"
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+0*int64(ext2.BlockSize), rootBlk); err != nil {
		return err
	}
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+1*int64(ext2.BlockSize), users); err != nil {
		return err
	}
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+2*int64(ext2.BlockSize), journal); err != nil {
		return err
	}

	// Actualiza contadores
	sb.SFreeInodesCount = sb.SInodesCount - 3
	sb.SFreeBlocksCount = sb.SBlocksCount - 3
	sb.SFirtsIno = ext2.FirstFree(bmIn)
	sb.SFirstBlo = ext2.FirstFree(bmBl)

	if err := writeAt(mp.DiskPath, partStart, sb); err != nil {
		return err
	}
	return nil
}
