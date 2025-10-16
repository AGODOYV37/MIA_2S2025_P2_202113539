package ext2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func AppendUsersLine(reg *mount.Registry, id, line string) error {

	cur, err := ReadUsersText(reg, id)
	if err != nil {
		return err
	}

	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return errors.New("append: línea vacía")
	}
	if !strings.HasSuffix(cur, "\n") && len(cur) > 0 {
		cur += "\n"
	}
	newContent := cur + line + "\n"

	return RewriteUsers(reg, id, newContent)
}

func RewriteUsers(reg *mount.Registry, id string, content string) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("users: id %s no está montado", id)
	}

	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("users: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("users: la partición no es EXT2 válida")
	}

	usersIdx, err := usersInodeIndex(mp, sb)
	if err != nil {
		return err
	}
	uino, err := readInodeAt(mp, sb, usersIdx)
	if err != nil {
		return fmt.Errorf("users: leyendo inodo users.txt: %w", err)
	}

	data := []byte(content)
	newSize := len(data)
	if newSize < 0 {
		newSize = 0
	}
	const B = BlockSize
	needBlocks := (newSize + B - 1) / B
	if needBlocks == 0 && newSize > 0 {
		needBlocks = 1
	}
	if needBlocks > InodeDirectCount {
		return fmt.Errorf("users: tamaño excede apuntadores directos (%d*%d)", InodeDirectCount, BlockSize)
	}

	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	var curBlocks []int32
	for _, ptr := range uino.IBlock {
		if ptr >= 0 {
			curBlocks = append(curBlocks, ptr)
		}
	}
	curCount := len(curBlocks)

	if needBlocks > curCount {
		add := needBlocks - curCount
		for i := 0; i < add; i++ {
			blkIdx := FirstFree(bmBl)
			if blkIdx < 0 {
				return errors.New("users: no hay bloques libres")
			}
			MarkBlock(bmBl, blkIdx, true)
			sb.SFreeBlocksCount--
			if uino.IBlock[curCount+i] != -1 && uino.IBlock[curCount+i] != 0 {

			}
			uino.IBlock[curCount+i] = blkIdx
			curBlocks = append(curBlocks, blkIdx)
		}
	}

	if needBlocks < curCount {
		for i := needBlocks; i < curCount; i++ {
			blk := uino.IBlock[i]
			if blk >= 0 {
				MarkBlock(bmBl, blk, false)
				sb.SFreeBlocksCount++
			}
			uino.IBlock[i] = -1
		}
		curBlocks = curBlocks[:needBlocks]
	}

	for i := 0; i < needBlocks; i++ {
		start := i * B
		end := start + B
		if end > newSize {
			end = newSize
		}
		var fblk BlockFile
		copy(fblk.BContent[:], make([]byte, B))
		copy(fblk.BContent[:], data[start:end])
		if err := writeFileBlockAt(mp, sb, curBlocks[i], fblk); err != nil {
			return err
		}
	}

	uino.ISize = int32(newSize)

	if err := writeInodeAt(mp, sb, usersIdx, uino); err != nil {
		return err
	}

	sb.SFirstBlo = FirstFree(bmBl)

	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	if err := writeAt(mp.DiskPath, mp.Start, sb); err != nil {
		return err
	}
	return nil
}

// ===== Helpers privados =====

func usersInodeIndex(mp *mount.MountedPartition, sb SuperBloque) (int32, error) {
	root, err := readInodeAt(mp, sb, 0)
	if err != nil {
		return -1, fmt.Errorf("users: leyendo inodo raíz: %w", err)
	}
	for _, ptr := range root.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return -1, err
		}
		for _, de := range bf.BContent {
			name := trimNull(de.BName[:])
			if name == "users.txt" && de.BInodo >= 0 {
				return de.BInodo, nil
			}
		}
	}
	// fallback de mkfs inicial
	return 1, nil
}
