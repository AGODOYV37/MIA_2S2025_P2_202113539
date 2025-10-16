package ext2

import (
	"errors"
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func MakeDir(reg *mount.Registry, id, absPath string, p bool, uid, gid int) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("mkdir: id %s no está montado", id)
	}

	// Leer superbloque
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("mkdir: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("mkdir: la partición no es EXT2 válida (SB)")
	}

	// Parsear ruta
	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("mkdir: path apunta a '/' (ya existe)")
	}
	parentComps := comps[:len(comps)-1]
	dirName := comps[len(comps)-1]

	if invalidName(dirName) || len(dirName) > 12 {
		return fmt.Errorf("mkdir: nombre de carpeta inválido (<=12, sin espacios/comas): %q", dirName)
	}

	// Bitmaps
	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	// Crear/asegurar padres
	parentIno, err := ensureDirPath(mp, &sb, bmIn, bmBl, parentComps, p, uid, gid)
	if err != nil {
		return err
	}

	exists := lookupInDir(mp, sb, parentIno, dirName)
	if exists >= 0 {
		ino, err := readInodeAt(mp, sb, exists)
		if err != nil {
			return err
		}
		if ino.IType != 0 {
			return fmt.Errorf("mkdir: '%s' ya existe y no es una carpeta", absPath)
		}
		// Es carpeta ya existente
		if p {
			return nil
		}
		return fmt.Errorf("mkdir: '%s' ya existe", absPath)
	}

	//  Crear nuevo inodo de carpeta
	inIdx := FirstFree(bmIn)
	if inIdx < 0 {
		return errors.New("mkdir: no hay inodos libres")
	}
	MarkInode(bmIn, inIdx, true)
	sb.SFreeInodesCount--

	// Reservar bloque de carpeta
	blk := FirstFree(bmBl)
	if blk < 0 {
		return errors.New("mkdir: no hay bloques libres")
	}
	MarkBlock(bmBl, blk, true)
	sb.SFreeBlocksCount--

	// Inodo carpeta
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
	fb.BContent[0].BInodo = inIdx
	copy(fb.BContent[1].BName[:], []byte(".."))
	fb.BContent[1].BInodo = parentIno

	if err := writeInodeAt(mp, sb, inIdx, dir); err != nil {
		return err
	}
	if err := writeFolderBlockAt(mp, sb, blk, fb); err != nil {
		return err
	}

	if err := addDirEntry(mp, &sb, bmBl, parentIno, dirName, inIdx); err != nil {
		return err
	}

	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, sb)
}
