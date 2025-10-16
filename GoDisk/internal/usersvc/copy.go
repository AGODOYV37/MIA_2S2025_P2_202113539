package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Copy(reg *mount.Registry, path, destino string) error {
	path = strings.TrimSpace(path)
	destino = strings.TrimSpace(destino)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("copy: -path debe ser absoluto")
	}
	if destino == "" || !strings.HasPrefix(destino, "/") {
		return errors.New("copy: -destino debe ser absoluto (carpeta existente)")
	}
	s, ok := auth.Current()
	if !ok {
		return errors.New("copy: requiere sesión (login)")
	}
	return ext2.CopyNode(reg, s.ID, path, destino, s.UID, s.GID, s.IsRoot)
}
