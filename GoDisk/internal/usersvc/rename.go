package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Rename(reg *mount.Registry, path, newName string) error {
	path = strings.TrimSpace(path)
	newName = strings.TrimSpace(newName)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("rename: -path inválido (debe ser absoluto)")
	}
	if newName == "" {
		return errors.New("rename: -name requerido")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("rename: requiere sesión (login)")
	}

	return ext2.RenameNode(reg, s.ID, path, newName, s.UID, s.GID, s.IsRoot)
}
