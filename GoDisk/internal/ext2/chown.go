package ext2

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// Chown cambia el propietario del nodo en startPath (y opcionalmente su contenido).
// Regla de permisos:
//   - root: siempre permitido.
//   - no root: solo puede cambiar nodos donde es propietario (IUid == uid).
//
// Recorrido (-r): solo entra a carpetas que puede leer (o si es root).
func Chown(reg *mount.Registry, id, startPath, newUser string, recursive bool, actorUID, actorGID int, isRoot bool) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return fmt.Errorf("chown: id %s no está montado", id)
	}
	var sb SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("chown: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemType || sb.SMagic != MagicEXT2 {
		return errors.New("chown: la partición no es EXT2 válida (SB)")
	}

	startPath = path.Clean(strings.TrimSpace(startPath))
	if !strings.HasPrefix(startPath, "/") {
		return errors.New("chown: -path debe ser absoluto")
	}
	if startPath == "/" {
		return errors.New("chown: no se permite usar '/'")
	}

	// Resolver inodo de inicio
	comps, err := splitPath(startPath)
	if err != nil {
		return err
	}
	startIno, exists, err := resolvePathInode(mp, sb, comps)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("chown: ruta no existe: %s", startPath)
	}

	// Resolver UID de 'newUser' desde users.txt
	newUID, err := lookupUserUID(reg, id, newUser)
	if err != nil {
		return err
	}

	// Cambia propietario de un inodo si procede según reglas
	changeOwner := func(idx int32, abs string) error {
		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			return err
		}
		// Root: siempre puede. No root: solo si es dueño actual.
		if !isRoot && int(ino.IUid) != actorUID {
			// avisamos y omitimos
			fmt.Printf("chown: omitido (no eres dueño): %s\n", abs)
			return nil
		}
		ino.IUid = int32(newUID)
		return writeInodeAt(mp, sb, idx, ino)
	}

	// Recorrido
	var walk func(idx int32, abs string) error
	walk = func(idx int32, abs string) error {
		ino, err := readInodeAt(mp, sb, idx)
		if err != nil {
			return err
		}
		// Primero cambiamos el propietario del nodo actual
		if err := changeOwner(idx, abs); err != nil {
			return err
		}
		// Si es carpeta y recursivo, entrar si tenemos lectura (o root)
		if ino.IType == 0 && recursive && CanRead(ino, actorUID, actorGID, isRoot) {
			entries, err := listDirEntries(mp, sb, idx)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if err := walk(e.ino, path.Join(abs, e.name)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return walk(startIno, startPath)
}

// lookupUserUID lee users.txt y devuelve el UID de un usuario activo.
// Error si no existe o está eliminado (UID=0).
func lookupUserUID(reg *mount.Registry, id, user string) (int, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return 0, errors.New("chown: -usuario vacío")
	}
	txt, err := ReadUsersText(reg, id)
	if err != nil {
		return 0, fmt.Errorf("chown: %w", err)
	}

	var uid int
	found := false
	for _, line := range strings.Split(strings.ReplaceAll(strings.ReplaceAll(txt, "\r\n", "\n"), "\r", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) == 5 && strings.EqualFold(parts[1], "U") {
			// form: "<uid>, U, <group>, <user>, <pass>"
			if strings.EqualFold(parts[3], user) {
				uid = atoi(parts[0])
				if uid == 0 {
					return 0, fmt.Errorf("chown: el usuario %q está eliminado", user)
				}
				found = true
				break
			}
		}
	}
	if !found {
		return 0, fmt.Errorf("chown: usuario %q no existe", user)
	}
	return uid, nil
}

func atoi(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		} else {
			break
		}
	}
	return n
}
