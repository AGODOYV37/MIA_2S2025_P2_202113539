package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
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

	s, err := auth.Require()
	if err != nil {
		return errors.New("copy: requiere sesión (login)")
	}

	if err := ext2.CopyNode(reg, s.ID, path, destino, s.UID, s.GID, s.IsRoot); err != nil {
		return err
	}

	_ = ext3.AppendJournalIfExt3(reg, s.ID, "COPY", path, "dest="+destino)
	return nil

}
