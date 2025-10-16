package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func Cat(reg *mount.Registry, files []string) (string, error) {
	if len(files) == 0 {
		return "", errors.New("cat: especifica al menos un -fileN")
	}
	s, ok := auth.Current()
	if !ok {
		return "", errors.New("cat: requiere sesión (login)")
	}

	var b strings.Builder
	for i, path := range files {
		path = strings.TrimSpace(path)
		if path == "" || !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("cat: ruta inválida: %q", path)
		}
		ino, data, err := ext2.ReadFileByPath(reg, s.ID, path)
		if err != nil {
			return "", err
		}
		if !canRead(ino, s) {
			return "", fmt.Errorf("cat: permiso denegado para %s", path)
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.Write(data)
	}
	return b.String(), nil
}

func canRead(ino ext2.Inodo, s *auth.Session) bool {
	if s.IsRoot {
		return true
	}
	var cls byte
	switch {
	case int32(s.UID) == ino.IUid:
		cls = 0
	case int32(s.GID) == ino.IGid:
		cls = 1
	default:
		cls = 2
	}
	perm := ino.IPerm[cls]
	return (perm & 0b100) != 0
}
