package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Edit(reg *mount.Registry, path string, data []byte) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("edit: -path inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("edit: requiere sesión (login)")
	}

	return ext2.EditFile(reg, s.ID, path, data, s.UID, s.GID, s.IsRoot)
}
