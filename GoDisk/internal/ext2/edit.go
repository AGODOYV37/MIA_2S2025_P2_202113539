package ext2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// EditFile reemplaza completamente el contenido de un archivo existente,
// validando permisos de lectura y escritura (o root).
func EditFile(reg *mount.Registry, id, absPath string, data []byte, uid, gid int, isRoot bool) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("edit: id %s no está montado", id)
	}

	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("edit: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("edit: la partición no es EXT2 válida (SB)")
	}

	// Parsear ruta: /a/b/c.txt  => parent=/a/b  name=c.txt
	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("edit: ruta apunta a '/' (no es archivo)")
	}
	parentComps := comps[:len(comps)-1]
	fileName := comps[len(comps)-1]

	// Navegar por los directorios existentes (no crea nada)
	parentIno, err := findDirPath(mp, sb, parentComps)
	if err != nil {
		return err
	}

	// Encontrar el archivo en el directorio padre
	childIno := lookupInDir(mp, sb, parentIno, fileName)
	if childIno < 0 {
		return fmt.Errorf("edit: %s no existe", absPath)
	}
	ino, err := readInodeAt(mp, sb, childIno)
	if err != nil {
		return err
	}
	if ino.IType != 1 {
		return fmt.Errorf("edit: %s no es un archivo", absPath)
	}

	// Permisos: requiere READ+WRITE (o root)
	if !canReadWrite(ino, uid, gid, isRoot) {
		return fmt.Errorf("edit: permisos insuficientes para '%s' (rw requeridos)", absPath)
	}

	// Escribir el nuevo contenido reaprovechando writeDataToFileInode
	bmIn, bmBl, err := loadBitmaps(mp, sb)
	if err != nil {
		return err
	}

	if err := writeDataToFileInode(mp, &sb, bmBl, childIno, data); err != nil {
		return err
	}

	// Guardar bitmaps y SB (inodos no cambian de asignación)
	sb.SFirtsIno = FirstFree(bmIn)
	sb.SFirstBlo = FirstFree(bmBl)
	if err := saveBitmaps(mp, sb, bmIn, bmBl); err != nil {
		return err
	}
	return writeAt(mp.DiskPath, mp.Start, sb)
}

// findDirPath navega por directorios (sin crear). Devuelve el inodo del último directorio.
func findDirPath(mp *mount.MountedPartition, sb SuperBloque, comps []string) (int32, error) {
	cur := int32(0)
	for i, name := range comps {
		next := lookupInDir(mp, sb, cur, name)
		if next < 0 {
			return -1, fmt.Errorf("edit: carpeta faltante '%s'", "/"+strings.Join(comps[:i+1], "/"))
		}
		ino, err := readInodeAt(mp, sb, next)
		if err != nil {
			return -1, err
		}
		if ino.IType != 0 {
			return -1, fmt.Errorf("edit: '%s' existe y no es carpeta", "/"+strings.Join(comps[:i+1], "/"))
		}
		cur = next
	}
	return cur, nil
}

// canReadWrite valida lectura y escritura según propietario/grupo/otros.
// Permisos codificados como sumas: read=4, write=2, exec=1.
func canReadWrite(ino Inodo, uid, gid int, isRoot bool) bool {
	if isRoot {
		return true
	}
	var p byte
	switch {
	case int(ino.IUid) == uid:
		p = ino.IPerm[0]
	case int(ino.IGid) == gid:
		p = ino.IPerm[1]
	default:
		p = ino.IPerm[2]
	}
	const R, W = 4, 2
	return (int(p)&R) != 0 && (int(p)&W) != 0
}
