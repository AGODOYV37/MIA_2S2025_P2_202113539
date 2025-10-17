package ext2

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func RenameNode(reg *mount.Registry, id, absPath, newName string, uid, gid int, isRoot bool) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("rename: id %s no está montado", id)
	}

	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("rename: leyendo SB: %w", err)
	}
	if err := requireSupportedFS(sb, "rename"); err != nil {
		return err
	}

	comps, err := splitPath(absPath)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("rename: no se puede renombrar '/'")
	}
	oldName := comps[len(comps)-1]
	if oldName == "." || oldName == ".." {
		return errors.New("rename: nombre especial no permitido")
	}
	if invalidRenameName(newName) {
		return fmt.Errorf("rename: nuevo nombre inválido (<=12, sin espacios/comas): %q", newName)
	}

	parentComps := comps[:len(comps)-1]
	parentIno, err := findDirPath(mp, sb, parentComps)
	if err != nil {
		return err
	}

	// localizar entrada actual
	childIno := lookupInDir(mp, sb, parentIno, oldName)
	if childIno < 0 {
		return fmt.Errorf("rename: '%s' no existe", absPath)
	}

	// verificar no colisión con el nuevo nombre en el mismo directorio
	if exists := lookupInDir(mp, sb, parentIno, newName); exists >= 0 {
		return fmt.Errorf("rename: ya existe '%s' en el mismo directorio", newName)
	}

	// permisos: escritura sobre el propio nodo (o root)
	ino, err := readInodeAt(mp, sb, childIno)
	if err != nil {
		return err
	}
	if !CanWrite(ino, uid, gid, isRoot) {
		return fmt.Errorf("rename: permisos insuficientes sobre '%s'", absPath)
	}

	// reemplazar el nombre en el directorio padre
	if err := replaceDirEntryName(mp, sb, parentIno, childIno, oldName, newName); err != nil {
		return err
	}
	return nil
}

func invalidRenameName(s string) bool {
	if len(s) == 0 || len(s) > 12 || strings.ContainsRune(s, ',') {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func replaceDirEntryName(mp *mount.MountedPartition, sb SuperBloque, parentIno, childIno int32, oldName, newName string) error {
	p, err := readInodeAt(mp, sb, parentIno)
	if err != nil {
		return err
	}
	for _, ptr := range p.IBlock {
		if ptr < 0 {
			continue
		}
		fb, err := readFolderBlockAt(mp, sb, ptr)
		if err != nil {
			return err
		}
		changed := false
		for i := range fb.BContent {
			nm := trimNull(fb.BContent[i].BName[:])
			if nm == oldName && fb.BContent[i].BInodo == childIno {
				// limpiar y escribir nuevo nombre
				for j := range fb.BContent[i].BName {
					fb.BContent[i].BName[j] = 0
				}
				copy(fb.BContent[i].BName[:], []byte(newName))
				changed = true
				break
			}
		}
		if changed {
			return writeFolderBlockAt(mp, sb, ptr, fb)
		}
	}
	return errors.New("rename: no se encontró la entrada del directorio (inconsistencia)")
}
