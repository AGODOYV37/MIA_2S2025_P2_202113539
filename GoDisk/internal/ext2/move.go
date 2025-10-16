package ext2

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func MoveNode(reg *mount.Registry, id, srcPath, destDir string, uid, gid int, isRoot bool) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("move: id %s no está montado", id)
	}
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("move: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("move: la partición no es EXT2 válida (SB)")
	}

	srcPath = path.Clean(strings.TrimSpace(srcPath))
	destDir = path.Clean(strings.TrimSpace(destDir))

	if !strings.HasPrefix(srcPath, "/") || !strings.HasPrefix(destDir, "/") {
		return errors.New("move: rutas deben ser absolutas")
	}
	if srcPath == "/" {
		return errors.New("move: no puedes mover '/'")
	}
	// Bloquea mover una carpeta dentro de sí misma o su subárbol por texto.
	// El sufijo "/" evita falsos positivos con archivos: "/a.txt" no matchea "/a.txt.bak".
	if destDir == srcPath || strings.HasPrefix(destDir, srcPath+"/") {
		return fmt.Errorf("move: no puedes mover %q dentro de %q", srcPath, destDir)
	}

	// Resolver origen
	srcComps, err := splitPath(srcPath)
	if err != nil {
		return err
	}
	if len(srcComps) == 0 {
		return errors.New("move: -path no puede ser '/'")
	}
	srcIno, exists, err := resolvePathInode(mp, sb, srcComps)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("move: origen no existe: %s", srcPath)
	}
	srcNode, err := readInodeAt(mp, sb, srcIno)
	if err != nil {
		return err
	}
	// Solo escritura sobre el ORIGEN
	if !canWrite(srcNode, uid, gid, isRoot) {
		return errors.New("move: sin permiso de escritura sobre el origen")
	}

	// Resolver padre de origen y nombre base
	parentComps := srcComps[:len(srcComps)-1]
	baseName := srcComps[len(srcComps)-1]
	srcParentIno, okParent, err := resolvePathInode(mp, sb, parentComps)
	if err != nil {
		return err
	}
	if !okParent {
		return errors.New("move: carpeta padre del origen no existe")
	}

	// Resolver destino (DEBE ser carpeta existente)
	dstComps, err := splitPath(destDir)
	if err != nil {
		return err
	}
	dstIno, dstExists, err := resolvePathInode(mp, sb, dstComps)
	if err != nil {
		return err
	}
	if !dstExists {
		return fmt.Errorf("move: carpeta destino no existe: %s", destDir)
	}
	dstNode, err := readInodeAt(mp, sb, dstIno)
	if err != nil {
		return err
	}
	if dstNode.IType != 0 {
		return fmt.Errorf("move: -destino debe ser una carpeta: %s", destDir)
	}

	// Evitar mover carpeta dentro de sí misma o de su propio subárbol
	if srcNode.IType == 0 {
		isDesc, err := isDescendant(mp, sb, srcIno, dstIno)
		if err != nil {
			return err
		}
		if isDesc {
			return errors.New("move: no puedes mover una carpeta a sí misma o dentro de su propio subárbol")
		}
	}

	// Colisión en destino
	if lookupInDir(mp, sb, dstIno, baseName) >= 0 {
		return fmt.Errorf("move: ya existe '%s' en '%s'", baseName, destDir)
	}

	// Cargar bitmaps (puede necesitar nuevo bloque en carpeta destino)
	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	// 1) Agregar entrada en destino
	if err := addDirEntry(mp, &sb, bmBl, dstIno, baseName, srcIno); err != nil {
		return err
	}

	// 2) Quitar entrada del viejo padre
	if err := removeDirEntry(mp, sb, srcParentIno, baseName); err != nil {
		return err
	}

	// Persistir cambios
	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, sb)
}

// -------- helpers específicos de move --------

// true si "candidate" está dentro del subárbol cuyo raíz es "ancestor"
func isDescendant(mp *mount.MountedPartition, sb SuperBloque, ancestor, candidate int32) (bool, error) {
	if ancestor == candidate {
		return true, nil
	}
	// subir desde candidate hasta root usando ".."
	cur := candidate
	limit := 1024 // corta ciclos anómalos
	for limit > 0 && cur != 0 {
		p, err := parentOf(mp, sb, cur)
		if err != nil {
			return false, err
		}
		if p == ancestor {
			return true, nil
		}
		if p == cur { // por si acaso
			break
		}
		cur = p
		limit--
	}
	return false, nil
}

func parentOf(mp *mount.MountedPartition, sb SuperBloque, dirIno int32) (int32, error) {
	ino, err := readInodeAt(mp, sb, dirIno)
	if err != nil {
		return -1, err
	}
	if ino.IType != 0 {
		return -1, errors.New("parentOf: no es carpeta")
	}
	first := ino.IBlock[0]
	if first < 0 {
		return -1, errors.New("parentOf: carpeta sin bloque 0")
	}
	bf, err := readFolderBlockAt(mp, sb, first)
	if err != nil {
		return -1, err
	}
	// convención: entry[1] = ".."
	return bf.BContent[1].BInodo, nil
}
