package ext2

import (
	"errors"
	"fmt"
	"path"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// --- Resolver una ruta absoluta (ya separada en comps) a índice de inodo.
// Devuelve (inoIdx, existe, error)
func resolvePathInode(mp *mount.MountedPartition, sb SuperBloque, comps []string) (int32, bool, error) {
	cur := int32(0) // raíz
	for _, name := range comps {
		next := lookupInDir(mp, sb, cur, name)
		if next < 0 {
			return -1, false, nil
		}
		cur = next
	}
	return cur, true, nil
}

// --- Leer todo el contenido (bytes) de un archivo por índice de inodo
func readDataFromFileInode(mp *mount.MountedPartition, sb SuperBloque, idx int32) ([]byte, error) {
	ino, err := readInodeAt(mp, sb, idx)
	if err != nil {
		return nil, err
	}
	if ino.IType != 1 {
		return nil, errors.New("readDataFromFileInode: inodo no es archivo")
	}
	sz := int(ino.ISize)
	if sz < 0 {
		sz = 0
	}
	out := make([]byte, 0, sz)
	rest := sz

	for _, p := range ino.IBlock {
		if p < 0 || rest <= 0 {
			break
		}
		bf, err := readFileBlockAt(mp, sb, p)
		if err != nil {
			return nil, err
		}
		n := BlockSize
		if n > rest {
			n = rest
		}
		out = append(out, bf.BContent[:n]...)
		rest -= n
	}
	return out, nil
}

// --- Estructura para enlistar hijos de un directorio
type dirChild struct {
	name  string
	ino   int32
	isDir bool
}

// --- Enumerar hijos visibles (omitiendo "." y "..")
func listDirEntries(mp *mount.MountedPartition, sb SuperBloque, dirIno int32) ([]dirChild, error) {
	ino, err := readInodeAt(mp, sb, dirIno)
	if err != nil {
		return nil, err
	}
	if ino.IType != 0 {
		return nil, errors.New("listDirEntries: no es carpeta")
	}

	var out []dirChild
	for _, ptr := range ino.IBlock {
		if ptr < 0 {
			continue
		}
		bf, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return nil, err
		}
		for _, e := range bf.BContent {
			if e.BInodo < 0 {
				continue
			}
			nm := trimNull(e.BName[:])
			if nm == "" || nm == "." || nm == ".." {
				continue
			}
			child, err := readInodeAt(mp, sb, e.BInodo)
			if err != nil {
				return nil, err
			}
			out = append(out, dirChild{
				name:  nm,
				ino:   e.BInodo,
				isDir: child.IType == 0,
			})
		}
	}
	return out, nil
}

// --- Copiar un archivo (srcIno) como NUEVO archivo dentro de (dstParentIno) con nombre dstName
// NO guarda bitmaps ni SB: el caller lo hace (para agrupar escrituras).
func copyFileToNew(mp *mount.MountedPartition, sb *SuperBloque, bmIn, bmBl []byte,
	srcIno, dstParentIno int32, dstName string, uid, gid int) error {

	srcNode, err := readInodeAt(mp, *sb, srcIno)
	if err != nil {
		return err
	}
	if srcNode.IType != 1 {
		return fmt.Errorf("copyFileToNew: origen no es archivo")
	}
	data, err := readDataFromFileInode(mp, *sb, srcIno)
	if err != nil {
		return err
	}

	// reservar inodo nuevo
	newIdx := FirstFree(bmIn)
	if newIdx < 0 {
		return errors.New("copy: no hay inodos libres para archivo")
	}
	MarkInode(bmIn, newIdx, true)
	sb.SFreeInodesCount--

	// crear inodo nuevo (mismos permisos del original)
	ino := newInodoArchivo(len(data))
	ino.IUid = int32(uid)
	ino.IGid = int32(gid)
	ino.IType = 1
	ino.IPerm = srcNode.IPerm
	for i := range ino.IBlock {
		if ino.IBlock[i] == 0 {
			ino.IBlock[i] = -1
		}
	}
	if err := writeInodeAt(mp, *sb, newIdx, ino); err != nil {
		return err
	}

	// escribir data (asigna bloques y descuenta del bitmap)
	if err := writeDataToFileInode(mp, sb, bmBl, newIdx, data); err != nil {
		return err
	}

	// enlazar en carpeta destino
	if err := addDirEntry(mp, sb, bmBl, dstParentIno, dstName, newIdx); err != nil {
		return err
	}
	return nil
}

// --- Utilidad para formar rutas de logging (evita //)
func joinAbs(base, name string) string {
	if base == "" || base == "/" {
		return "/" + name
	}
	return path.Join("/", base, name)
}
