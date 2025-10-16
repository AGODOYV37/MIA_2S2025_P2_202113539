package ext2

import (
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

type Formatter struct {
	reg *mount.Registry
}

func NewFormatter(reg *mount.Registry) *Formatter { return &Formatter{reg: reg} }

func (f *Formatter) MkfsFull(id string) error {
	mp, ok := f.reg.GetByID(id)
	if !ok {
		return fmt.Errorf("mkfs: id %s no está montado", id)
	}

	partStart := mp.Start
	partSize := mp.Size

	_, sb, err := ComputeLayout(partSize)
	if err != nil {
		return err
	}

	if err := writeAt(mp.DiskPath, partStart, sb); err != nil {
		return fmt.Errorf("mkfs: error escribiendo superbloque: %w", err)
	}

	bmIn, bmBl := NewBitmaps(sb.SInodesCount, sb.SBlocksCount)

	MarkInode(bmIn, 0, true)
	MarkBlock(bmBl, 0, true)
	MarkInode(bmIn, 1, true)
	MarkBlock(bmBl, 1, true)

	if err := writeBytes(mp.DiskPath, partStart+sb.SBmInodeStart, bmIn); err != nil {
		return err
	}
	if err := writeBytes(mp.DiskPath, partStart+sb.SBmBlockStart, bmBl); err != nil {
		return err
	}

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

	rootBlk := buildRootBlock()
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+0*int64(BlockSize), rootBlk); err != nil {
		return err
	}
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+1*int64(BlockSize), users); err != nil {
		return err
	}

	sb.SFreeInodesCount = sb.SInodesCount - 2
	sb.SFreeBlocksCount = sb.SBlocksCount - 2
	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := writeAt(mp.DiskPath, partStart, sb); err != nil {
		return err
	}

	return nil
}
