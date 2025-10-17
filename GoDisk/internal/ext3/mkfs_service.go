package ext3

import (
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

type Formatter struct{ reg *mount.Registry }

func NewFormatter(reg *mount.Registry) *Formatter { return &Formatter{reg: reg} }

func (f *Formatter) MkfsFull(id string) error {
	mp, ok := f.reg.GetByID(id)
	if !ok {
		return fmt.Errorf("mkfs: id %s no está montado", id)
	}
	partStart := mp.Start
	partSize := mp.Size

	// Layout EXT3 con journaling como región (n * 50 bytes)
	_, sb, jOff, jLen, err := ComputeLayoutExt3(partSize)
	if err != nil {
		return err
	}

	// Escribe SB
	if err := writeAt(mp.DiskPath, partStart, sb); err != nil {
		return fmt.Errorf("mkfs: error escribiendo superbloque: %w", err)
	}

	// Inicializa el área de journal (cero)
	if err := writeBytes(mp.DiskPath, partStart+jOff, make([]byte, jLen)); err != nil {
		return fmt.Errorf("mkfs: inicializando journaling: %w", err)
	}

	// Bitmaps
	bmIn, bmBl := ext2.NewBitmaps(sb.SInodesCount, sb.SBlocksCount)

	// Reservar root (0) y users.txt (1)
	ext2.MarkInode(bmIn, 0, true)
	ext2.MarkBlock(bmBl, 0, true)
	ext2.MarkInode(bmIn, 1, true)
	ext2.MarkBlock(bmBl, 1, true)

	// Persistir bitmaps
	if err := writeBytes(mp.DiskPath, partStart+sb.SBmInodeStart, bmIn); err != nil {
		return err
	}
	if err := writeBytes(mp.DiskPath, partStart+sb.SBmBlockStart, bmBl); err != nil {
		return err
	}

	// Inodos iniciales
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

	// Bloque de carpeta raíz: ".", "..", "users.txt"
	var rootBlk ext2.BlockFolder
	copy(rootBlk.BContent[0].BName[:], []byte("."))
	rootBlk.BContent[0].BInodo = 0
	copy(rootBlk.BContent[1].BName[:], []byte(".."))
	rootBlk.BContent[1].BInodo = 0
	copy(rootBlk.BContent[2].BName[:], []byte("users.txt"))
	rootBlk.BContent[2].BInodo = 1

	// Escribir bloques: root folder y users.txt
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+0*int64(ext2.BlockSize), rootBlk); err != nil {
		return err
	}
	if err := writeAt(mp.DiskPath, partStart+sb.SBlockStart+1*int64(ext2.BlockSize), users); err != nil {
		return err
	}

	// Actualiza contadores y punteros libres
	sb.SFreeInodesCount = sb.SInodesCount - 2
	sb.SFreeBlocksCount = sb.SBlocksCount - 2
	sb.SFirtsIno = ext2.FirstFree(bmIn)
	sb.SFirstBlo = ext2.FirstFree(bmBl)

	return writeAt(mp.DiskPath, partStart, sb)
}
