package ext3

import (
	"errors"
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// zeroRegion escribe ceros en 'length' bytes comenzando en 'off'.
// Lo hace por bloques para evitar grandes asignaciones.
func zeroRegion(diskPath string, off, length int64) error {
	const chunk = 1 << 20 // 1 MiB
	buf := make([]byte, chunk)
	var written int64
	for written < length {
		n := length - written
		if n > chunk {
			n = chunk
		}
		if err := writeBytes(diskPath, off+written, buf[:n]); err != nil {
			return err
		}
		written += n
	}
	return nil
}

// Loss "rompe" el FS EXT3 poniendo a cero:
// - Bitmap de inodos
// - Bitmap de bloques
// - Área de tabla de inodos
// - Área de bloques de datos
// No toca SB ni el área de journal.
func Loss(reg *mount.Registry, id string) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("loss: id %s no está montado", id)
	}

	// Leer SB
	var sb ext2.SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("loss: leyendo SB: %w", err)
	}

	// Solo aplicamos a EXT3 (el objetivo es simular pérdida y luego probar recovery)
	if sb.SFilesystemType != FileSystemTypeExt3 {
		return errors.New("loss: solo aplica para particiones EXT3")
	}

	// Offsets absolutos en el archivo de disco
	bmInOff := mp.Start + sb.SBmInodeStart
	bmBlOff := mp.Start + sb.SBmBlockStart
	inTblOff := mp.Start + sb.SInodeStart
	blkTblOff := mp.Start + sb.SBlockStart

	// Tamaños
	bmInLen := int64(sb.SInodesCount)                           // 1 byte por inodo
	bmBlLen := int64(sb.SBlocksCount)                           // 1 byte por bloque (3n)
	inTblLen := int64(sb.SInodesCount) * int64(sb.SInodeS)      // tabla de inodos
	blkTblLen := int64(sb.SBlocksCount) * int64(ext2.BlockSize) // área de bloques

	// ZEROS
	if err := zeroRegion(mp.DiskPath, bmInOff, bmInLen); err != nil {
		return fmt.Errorf("loss: limpiando bitmap de inodos: %w", err)
	}
	if err := zeroRegion(mp.DiskPath, bmBlOff, bmBlLen); err != nil {
		return fmt.Errorf("loss: limpiando bitmap de bloques: %w", err)
	}
	if err := zeroRegion(mp.DiskPath, inTblOff, inTblLen); err != nil {
		return fmt.Errorf("loss: limpiando área de inodos: %w", err)
	}
	if err := zeroRegion(mp.DiskPath, blkTblOff, blkTblLen); err != nil {
		return fmt.Errorf("loss: limpiando área de bloques: %w", err)
	}

	return nil
}
