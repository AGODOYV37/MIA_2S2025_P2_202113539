package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func Mkdir(reg *mount.Registry, path string, p bool) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("mkdir: -path inválido (debe ser absoluto)")
	}
	s, ok := auth.Current()
	if !ok {
		return errors.New("mkdir: requiere sesión (login)")
	}
	return ext2.MakeDir(reg, s.ID, path, p, s.UID, s.GID)
}
