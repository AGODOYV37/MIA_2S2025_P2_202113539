package ext2

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// ========== Lectura / Escritura cruda en offset ==========

func readAt(path string, off int64, data any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	return binary.Read(f, binary.LittleEndian, data)
}

func writeAt(path string, off int64, data any) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, data)
}

func readBytes(path string, off int64, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(f, buf)
	return buf, err
}

func writeBytes(path string, off int64, buf []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	_, err = f.Write(buf)
	return err
}

// ========== Lectura / Escritura de estructuras EXT2 ==========

func readInodeAt(mp *mount.MountedPartition, sb SuperBloque, idx int32) (Inodo, error) {
	var ino Inodo
	off := mp.Start + sb.SInodeStart + int64(idx)*int64(sb.SInodeS)
	if err := readAt(mp.DiskPath, off, &ino); err != nil {
		return Inodo{}, err
	}
	return ino, nil
}

func writeInodeAt(mp *mount.MountedPartition, sb SuperBloque, idx int32, ino Inodo) error {
	off := mp.Start + sb.SInodeStart + int64(idx)*int64(sb.SInodeS)
	return writeAt(mp.DiskPath, off, ino)
}

func readFolderBlockAt(mp *mount.MountedPartition, sb SuperBloque, blk int32) (BlockFolder, error) {
	var b BlockFolder
	off := mp.Start + sb.SBlockStart + int64(blk)*int64(BlockSize)
	if err := readAt(mp.DiskPath, off, &b); err != nil {
		return BlockFolder{}, err
	}
	return b, nil
}

func writeFolderBlockAt(mp *mount.MountedPartition, sb SuperBloque, blk int32, b BlockFolder) error {
	off := mp.Start + sb.SBlockStart + int64(blk)*int64(BlockSize)
	return writeAt(mp.DiskPath, off, b)
}

func readFileBlockAt(mp *mount.MountedPartition, sb SuperBloque, blk int32) (BlockFile, error) {
	var b BlockFile
	off := mp.Start + sb.SBlockStart + int64(blk)*int64(BlockSize)
	if err := readAt(mp.DiskPath, off, &b); err != nil {
		return BlockFile{}, err
	}
	return b, nil
}

func writeFileBlockAt(mp *mount.MountedPartition, sb SuperBloque, blk int32, b BlockFile) error {
	off := mp.Start + sb.SBlockStart + int64(blk)*int64(BlockSize)
	return writeAt(mp.DiskPath, off, b)
}

// ========== Bitmaps (modelo 1 byte por entrada) ==========

func loadBitmaps(mp *mount.MountedPartition, sb SuperBloque) ([]byte, []byte, error) {
	szIn := int(sb.SInodesCount)
	szBl := int(sb.SBlocksCount)

	bmIn, err := readBytes(mp.DiskPath, mp.Start+sb.SBmInodeStart, szIn)
	if err != nil {
		return nil, nil, fmt.Errorf("ext2: leyendo bm_inode: %w", err)
	}
	bmBl, err := readBytes(mp.DiskPath, mp.Start+sb.SBmBlockStart, szBl)
	if err != nil {
		return nil, nil, fmt.Errorf("ext2: leyendo bm_block: %w", err)
	}
	return bmIn, bmBl, nil
}

func saveBitmaps(mp *mount.MountedPartition, sb SuperBloque, bmIn, bmBl []byte) error {
	if err := writeBytes(mp.DiskPath, mp.Start+sb.SBmInodeStart, bmIn); err != nil {
		return fmt.Errorf("ext2: escribiendo bm_inode: %w", err)
	}
	if err := writeBytes(mp.DiskPath, mp.Start+sb.SBmBlockStart, bmBl); err != nil {
		return fmt.Errorf("ext2: escribiendo bm_block: %w", err)
	}
	return nil
}

// ========== Utilidad menor ==========

func trimNull(b []byte) string {
	i := len(b)
	for i > 0 && b[i-1] == 0 {
		i--
	}
	return strings.TrimSpace(string(b[:i]))
}
