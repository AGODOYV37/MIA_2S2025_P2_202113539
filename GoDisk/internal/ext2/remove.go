package ext2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Remove(reg *mount.Registry, id, absPath string, uid, gid int) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("remove: id %s no está montado", id)
	}
	if absPath == "/" {
		return errors.New("remove: no se puede eliminar '/'")
	}

	// SB
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("remove: leyendo SB: %w", err)
	}
	if err := requireSupportedFS(sb, "remove"); err != nil {
		return err
	}

	// Parse path
	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("remove: path apunta a '/'")
	}
	parentComps := comps[:len(comps)-1]
	targetName := comps[len(comps)-1]

	// Cargar bitmaps
	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	// Resolver padre (debe existir, no crear)
	parentIno, err := walkExistingDirPath(mp, &sb, parentComps)
	if err != nil {
		return err
	}

	// Resolver objetivo
	targetIno := lookupInDir(mp, sb, parentIno, targetName)
	if targetIno < 0 {
		return fmt.Errorf("remove: no existe %q", absPath)
	}
	tIno, err := readInodeAt(mp, sb, targetIno)
	if err != nil {
		return err
	}

	// Pre-chequeo de permisos en TODO el subárbol
	if ok := subtreeWritable(mp, sb, targetIno, uid, gid); !ok {
		return fmt.Errorf("remove: permiso denegado en algún elemento dentro de %q", absPath)
	}

	//  Borrado recursivo real
	if err := deleteNode(mp, &sb, bmIn, bmBl, parentIno, targetName, targetIno, tIno); err != nil {
		return err
	}

	// Persistir bitmaps y SB
	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, sb)
}

// Recorre un path de carpetas que deben existir
func walkExistingDirPath(mp *mount.MountedPartition, sb *SuperBloque, comps []string) (int32, error) {
	cur := int32(0)
	for i, name := range comps {
		next := lookupInDir(mp, *sb, cur, name)
		if next < 0 {
			return -1, fmt.Errorf("remove: carpeta faltante '%s'", strings.Join(comps[:i+1], "/"))
		}
		ino, err := readInodeAt(mp, *sb, next)
		if err != nil {
			return -1, err
		}
		if ino.IType != 0 {
			return -1, fmt.Errorf("remove: '%s' no es carpeta", strings.Join(comps[:i+1], "/"))
		}
		cur = next
	}
	return cur, nil
}

func subtreeWritable(mp *mount.MountedPartition, sb SuperBloque, idx int32, uid, gid int) bool {
	ino, err := readInodeAt(mp, sb, idx)
	if err != nil {
		return false
	}
	if !hasWrite(uid, gid, ino) {
		return false
	}
	if ino.IType == 1 {
		return true // archivo
	}
	// carpeta: validar hijos
	children, err := listDirChildren(mp, sb, idx)
	if err != nil {
		return false
	}
	for _, ch := range children {
		if !subtreeWritable(mp, sb, ch.Ino, uid, gid) {
			return false
		}
	}
	return true
}

func hasWrite(uid, gid int, ino Inodo) bool {

	if uid == 1 {
		return true
	}
	var p byte
	switch {
	case uid == int(ino.IUid):
		p = ino.IPerm[0]
	case gid == int(ino.IGid):
		p = ino.IPerm[1]
	default:
		p = ino.IPerm[2]
	}
	return (p & 2) != 0 // bit de escritura
}

type childEntry struct {
	Name string
	Ino  int32
}

func listDirChildren(mp *mount.MountedPartition, sb SuperBloque, dirIno int32) ([]childEntry, error) {
	ino, err := readInodeAt(mp, sb, dirIno)
	if err != nil {
		return nil, err
	}
	var out []childEntry
	for _, ptr := range ino.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return nil, err
		}
		for _, e := range bf.BContent {
			nm := trimNull(e.BName[:])
			if nm == "" || nm == "." || nm == ".." || e.BInodo < 0 {
				continue
			}
			out = append(out, childEntry{Name: nm, Ino: e.BInodo})
		}
	}
	return out, nil
}

func deleteNode(mp *mount.MountedPartition, sb *SuperBloque, bmIn, bmBl []byte, parentIno int32, name string, idx int32, ino Inodo) error {

	if ino.IType == 0 {
		children, err := listDirChildren(mp, *sb, idx)
		if err != nil {
			return err
		}
		for _, ch := range children {
			cIno, err := readInodeAt(mp, *sb, ch.Ino)
			if err != nil {
				return err
			}
			if err := deleteNode(mp, sb, bmIn, bmBl, idx, ch.Name, ch.Ino, cIno); err != nil {
				return err
			}
		}

		for i, ptr := range ino.IBlock {
			if ptr >= 0 {
				MarkBlock(bmBl, ptr, false)
				sb.SFreeBlocksCount++
				ino.IBlock[i] = -1
			}
		}
		ino.ISize = 0
		if err := writeInodeAt(mp, *sb, idx, ino); err != nil {
			return err
		}
	} else {

		if err := writeDataToFileInode(mp, sb, bmBl, idx, []byte{}); err != nil {
			return err
		}
	}

	if err := removeDirEntry(mp, *sb, parentIno, name); err != nil {
		return err
	}

	//  Liberar inodo
	MarkInode(bmIn, idx, false)
	sb.SFreeInodesCount++

	// ( limpiar el inodo en disco
	var zero Inodo
	if err := writeInodeAt(mp, *sb, idx, zero); err != nil {
		return err
	}
	return nil
}
