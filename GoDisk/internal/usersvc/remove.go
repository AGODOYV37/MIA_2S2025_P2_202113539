package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Remove(reg *mount.Registry, path string) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("remove: -path inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("remove: requiere sesión (login)")
	}

	return ext2.Remove(reg, s.ID, path, s.UID, s.GID)
}
