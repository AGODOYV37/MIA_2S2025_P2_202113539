package ext2

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CopyNode(reg *mount.Registry, id, srcPath, destDir string, uid, gid int, isRoot bool) error {
	srcPath = strings.TrimSpace(srcPath)
	destDir = strings.TrimSpace(destDir)

	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("copy: id %s no está montado", id)
	}
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("copy: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("copy: la partición no es EXT2 válida (SB)")
	}

	// --- Origen ---
	srcComps, err := splitPath(srcPath)
	if err != nil {
		return err
	}
	if len(srcComps) == 0 {
		return errors.New("copy: -path no puede ser '/'")
	}
	srcIno, exists, err := resolvePathInode(mp, sb, srcComps)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("copy: ruta origen no existe: %s", srcPath)
	}
	srcNode, err := readInodeAt(mp, sb, srcIno)
	if err != nil {
		return err
	}
	if !canRead(srcNode, uid, gid, isRoot) {
		fmt.Printf("copy: sin permiso de lectura sobre '%s' (omitido)\n", srcPath)
		return nil
	}

	// --- Destino (carpeta existente y escribible) ---
	dstComps, err := splitPath(destDir)
	if err != nil {
		return err
	}
	dstIno, exists, err := resolvePathInode(mp, sb, dstComps)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("copy: carpeta destino no existe: %s", destDir)
	}
	dstNode, err := readInodeAt(mp, sb, dstIno)
	if err != nil {
		return err
	}
	if dstNode.IType != 0 {
		return fmt.Errorf("copy: -destino debe ser una carpeta: %s", destDir)
	}
	if !canWrite(dstNode, uid, gid, isRoot) {
		return fmt.Errorf("copy: sin permiso de escritura en carpeta destino")
	}

	// --- Evitar copiar en sí mismo o subcarpeta propia ---
	clean := func(comps []string) string {
		s := "/" + strings.Trim(strings.Join(comps, "/"), "/")
		if s == "" {
			return "/"
		}
		return s
	}
	srcAbs := clean(srcComps)
	dstAbs := clean(dstComps)
	if dstAbs == srcAbs || strings.HasPrefix(dstAbs+"/", srcAbs+"/") {
		return fmt.Errorf("copy: destino %q está dentro del origen %q", destDir, srcPath)
	}

	baseName := srcComps[len(srcComps)-1]
	if existing := lookupInDir(mp, sb, dstIno, baseName); existing >= 0 {
		fmt.Printf("copy: '%s' ya existe dentro de '%s' (omitido)\n", baseName, destDir)
		return nil
	}

	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	if srcNode.IType == 1 {
		// Archivo
		if err := copyFileToNew(mp, &sb, bmIn, bmBl, srcIno, dstIno, baseName, uid, gid); err != nil {
			return err
		}
		sb.SFirtsIno = FirstFree(bmIn)
		sb.SFirstBlo = FirstFree(bmBl)
		if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
			return err
		}
		return writeAt(mp.DiskPath, mp.Start, sb)
	}

	// Carpeta (recursivo, omite hijos sin permisos)
	if err := copyDirToNewSkip(mp, &sb, bmIn, bmBl, srcIno, dstIno, baseName, uid, gid, isRoot, strings.TrimPrefix(srcAbs, "/")); err != nil {
		return err
	}
	return nil
}

// Variante de copyDirToNew que **omite** hijos sin permisos y continúa
func copyDirToNewSkip(mp *mount.MountedPartition, sb *SuperBloque, bmIn, bmBl []byte, srcIno, dstParentIno int32, dstName string, uid, gid int, isRoot bool, srcAbs string) error {
	// crear carpeta destino
	newIdx := FirstFree(bmIn)
	if newIdx < 0 {
		return errors.New("copy: no hay inodos libres para carpeta")
	}
	MarkInode(bmIn, newIdx, true)
	sb.SFreeInodesCount--

	blk := FirstFree(bmBl)
	if blk < 0 {
		return errors.New("copy: no hay bloques libres para carpeta")
	}
	MarkBlock(bmBl, blk, true)
	sb.SFreeBlocksCount--

	srcNode, _ := readInodeAt(mp, *sb, srcIno)
	dir := newInodoCarpeta()
	dir.IUid = int32(uid)
	dir.IGid = int32(gid)
	dir.IType = 0
	dir.IPerm = srcNode.IPerm
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
	fb.BContent[1].BInodo = dstParentIno

	if err := writeInodeAt(mp, *sb, newIdx, dir); err != nil {
		return err
	}
	if err := writeFolderBlockAt(mp, *sb, blk, fb); err != nil {
		return err
	}
	if err := addDirEntry(mp, sb, bmBl, dstParentIno, dstName, newIdx); err != nil {
		return err
	}

	// copiar hijos (omitimos sin permiso)
	children, err := listDirEntries(mp, *sb, srcIno)
	if err != nil {
		return err
	}
	for _, ch := range children {
		chNode, err := readInodeAt(mp, *sb, ch.ino)
		if err != nil {
			return err
		}
		srcChildAbs := path.Join("/", srcAbs, ch.name)
		if !canRead(chNode, uid, gid, isRoot) {
			fmt.Printf("copy: sin permiso de lectura sobre '%s' (omitido)\n", srcChildAbs)
			continue
		}
		if ch.isDir {
			if err := copyDirToNewSkip(mp, sb, bmIn, bmBl, ch.ino, newIdx, ch.name, uid, gid, isRoot, srcChildAbs); err != nil {
				return err
			}
		} else {
			// ¿colisión dentro del nuevo directorio?
			if lookupInDir(mp, *sb, newIdx, ch.name) >= 0 {
				fmt.Printf("copy: '%s' ya existe dentro de '%s/%s' (omitido)\n", ch.name, srcAbs, dstName)
				continue
			}
			if err := copyFileToNew(mp, sb, bmIn, bmBl, ch.ino, newIdx, ch.name, uid, gid); err != nil {
				return err
			}
		}
	}

	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, *sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, *sb)
}
