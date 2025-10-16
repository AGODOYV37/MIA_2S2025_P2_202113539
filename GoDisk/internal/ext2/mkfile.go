package ext2

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CreateOrOverwriteFile(reg *mount.Registry, id, absPath string, data []byte, recursive, force bool, uid, gid int) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("mkfile: id %s no está montado", id)
	}

	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("mkfile: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("mkfile: la partición no es EXT2 válida (SB)")
	}

	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("mkfile: path apunta a '/' (no es archivo)")
	}
	parentComps := comps[:len(comps)-1]
	fileName := comps[len(comps)-1]

	if invalidName(fileName) || len(fileName) > 12 {
		return fmt.Errorf("mkfile: nombre de archivo inválido (<=12, sin espacios/comas): %q", fileName)
	}

	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	parentIno, err := ensureDirPath(mp, &sb, bmIn, bmBl, parentComps, recursive, uid, gid)
	if err != nil {
		return err
	}

	childIno := lookupInDir(mp, sb, parentIno, fileName)
	if childIno >= 0 {
		if !force {
			return fmt.Errorf("mkfile: %s ya existe; usa -force para sobreescribir", absPath)
		}

		if err := writeDataToFileInode(mp, &sb, bmBl, childIno, data); err != nil {
			return err
		}

		if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
			return err
		}
		return writeAt(mp.DiskPath, mp.Start, sb)
	}

	inIdx := FirstFree(bmIn)
	if inIdx < 0 {
		return errors.New("mkfile: no hay inodos libres")
	}
	MarkInode(bmIn, inIdx, true)
	sb.SFreeInodesCount--

	ino := newInodoArchivo(len(data))
	ino.IUid = int32(uid)
	ino.IGid = int32(gid)
	ino.IType = 1
	ino.IPerm = [3]byte{6, 6, 4}
	for i := range ino.IBlock {
		if ino.IBlock[i] == 0 {
			ino.IBlock[i] = -1
		}
	}

	if err := writeInodeAt(mp, sb, inIdx, ino); err != nil {
		return err
	}

	if err := writeDataToFileInode(mp, &sb, bmBl, inIdx, data); err != nil {
		return err
	}
	if err := writeDataToFileInode(mp, &sb, bmBl, inIdx, data); err != nil {
		return err
	}

	if err := writeInodeAt(mp, sb, inIdx, ino); err != nil {
		return err
	}

	if err := addDirEntry(mp, &sb, bmBl, parentIno, fileName, inIdx); err != nil {
		return err
	}

	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, sb)
}

func splitPath(p string) ([]string, error) {
	if p == "" || p[0] != '/' {
		return nil, errors.New("ruta debe ser absoluta")
	}
	if p == "/" {
		return []string{}, nil
	}
	items := strings.Split(p, "/")
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it == "" {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func invalidName(s string) bool {
	if strings.ContainsRune(s, ',') {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func ensureDirPath(mp *mount.MountedPartition, sb *SuperBloque, bmIn, bmBl []byte, comps []string, recursive bool, uid, gid int) (int32, error) {
	cur := int32(0)
	for i, name := range comps {
		if invalidName(name) || len(name) > 12 {
			return -1, fmt.Errorf("nombre de carpeta inválido (<=12): %q", name)
		}
		next := lookupInDir(mp, *sb, cur, name)
		if next >= 0 {

			ino, err := readInodeAt(mp, *sb, next)
			if err != nil {
				return -1, err
			}
			if ino.IType != 0 {
				return -1, fmt.Errorf("'%s' existe y no es carpeta", strings.Join(comps[:i+1], "/"))
			}
			cur = next
			continue
		}

		if !recursive {
			return -1, fmt.Errorf("carpeta faltante '%s'; usa -r para crear padres", strings.Join(comps[:i+1], "/"))
		}
		// crear carpeta
		newIdx := FirstFree(bmIn)
		if newIdx < 0 {
			return -1, errors.New("sin inodos libres para carpeta")
		}
		MarkInode(bmIn, newIdx, true)
		sb.SFreeInodesCount--

		// reservar bloque para carpeta
		blk := FirstFree(bmBl)
		if blk < 0 {
			return -1, errors.New("sin bloques libres para carpeta")
		}
		MarkBlock(bmBl, blk, true)
		sb.SFreeBlocksCount--

		// inodo carpeta
		dir := newInodoCarpeta()
		dir.IUid = int32(uid)
		dir.IGid = int32(gid)
		dir.IType = 0
		dir.IPerm = [3]byte{6, 6, 4}
		for i := range dir.IBlock {
			if dir.IBlock[i] == 0 {
				dir.IBlock[i] = -1
			}
		}
		dir.IBlock[0] = blk
		dir.ISize += int32(BlockSize)

		var fb BlockFolder
		copy(fb.BContent[0].BName[:], []byte("."))
		fb.BContent[0].BInodo = newIdx
		copy(fb.BContent[1].BName[:], []byte(".."))
		fb.BContent[1].BInodo = cur

		if err := writeInodeAt(mp, *sb, newIdx, dir); err != nil {
			return -1, err
		}
		if err := writeFolderBlockAt(mp, *sb, blk, fb); err != nil {
			return -1, err
		}

		// enlazar en el padre
		if err := addDirEntry(mp, sb, bmBl, cur, name, newIdx); err != nil {
			return -1, err
		}
		cur = newIdx
	}
	return cur, nil
}

func lookupInDir(mp *mount.MountedPartition, sb SuperBloque, dirIno int32, name string) int32 {
	ino, err := readInodeAt(mp, sb, dirIno)
	if err != nil {
		return -1
	}
	for _, ptr := range ino.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return -1
		}
		for _, e := range bf.BContent {
			nm := trimNull(e.BName[:])
			if nm == name && e.BInodo >= 0 {
				return e.BInodo
			}
		}
	}
	return -1
}

func addDirEntry(mp *mount.MountedPartition, sb *SuperBloque, bmBl []byte, dirIno int32, name string, childIno int32) error {
	ino, err := readInodeAt(mp, *sb, dirIno)
	if err != nil {
		return err
	}

	for bi, ptr := range ino.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, *sb, ptr)
		if err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if trimNull(bf.BContent[i].BName[:]) == "" {
				copy(bf.BContent[i].BName[:], []byte(name))
				bf.BContent[i].BInodo = childIno
				return writeFolderBlockAt(mp, *sb, ptr, bf)
			}
		}
		_ = bi
	}

	newBlk := FirstFree(bmBl)
	if newBlk < 0 {
		return errors.New("addDirEntry: no hay bloques libres")
	}
	MarkBlock(bmBl, newBlk, true)
	sb.SFreeBlocksCount--

	var fb BlockFolder
	copy(fb.BContent[0].BName[:], []byte(name))
	fb.BContent[0].BInodo = childIno

	for i := range ino.IBlock {
		if ino.IBlock[i] < 0 {
			ino.IBlock[i] = newBlk
			ino.ISize += int32(BlockSize)
			if err := writeInodeAt(mp, *sb, dirIno, ino); err != nil {
				return err
			}
			return writeFolderBlockAt(mp, *sb, newBlk, fb)
		}
	}
	return errors.New("addDirEntry: sin punteros directos libres en directorio")
}

func writeDataToFileInode(mp *mount.MountedPartition, sb *SuperBloque, bmBl []byte, idx int32, data []byte) error {
	ino, err := readInodeAt(mp, *sb, idx)
	if err != nil {
		return err
	}

	for i := range ino.IBlock {
		if ino.IBlock[i] == 0 {
			ino.IBlock[i] = -1
		}
	}
	want := (len(data) + BlockSize - 1) / BlockSize
	if want == 0 && len(data) > 0 {
		want = 1
	}

	// bloques actuales
	var cur []int32
	for _, p := range ino.IBlock {
		if p >= 0 {
			cur = append(cur, p)
		}
	}

	if want > len(cur) {
		add := want - len(cur)
		for i := 0; i < add; i++ {
			b := FirstFree(bmBl)
			if b < 0 {
				return errors.New("mkfile: sin bloques libres para archivo")
			}
			MarkBlock(bmBl, b, true)
			sb.SFreeBlocksCount--
			cur = append(cur, b)
		}

		w := 0
		for i := range ino.IBlock {
			if ino.IBlock[i] < 0 {
				ino.IBlock[i] = cur[len(cur)-add+w]
				w++
				if w == add {
					break
				}
			}
		}
	}

	if want < len(cur) {
		for i := want; i < len(cur); i++ {
			MarkBlock(bmBl, cur[i], false)
			sb.SFreeBlocksCount++
		}
		cur = cur[:want]

		cnt := 0
		for i := range ino.IBlock {
			if cnt >= want {
				ino.IBlock[i] = -1
			} else if ino.IBlock[i] >= 0 {
				cnt++
			}
		}
	}

	for i := 0; i < want; i++ {
		start := i * BlockSize
		end := start + BlockSize
		if end > len(data) {
			end = len(data)
		}
		var bf BlockFile
		copy(bf.BContent[:], make([]byte, BlockSize))
		copy(bf.BContent[:], data[start:end])
		if err := writeFileBlockAt(mp, *sb, cur[i], bf); err != nil {
			return err
		}
	}
	ino.ISize = int32(len(data))
	return writeInodeAt(mp, *sb, idx, ino)
}
