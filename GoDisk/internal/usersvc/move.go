package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Move(reg *mount.Registry, src, dst string) error {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(dst)
	if src == "" || !strings.HasPrefix(src, "/") {
		return errors.New("move: -path inválido (debe ser absoluto)")
	}
	if dst == "" || !strings.HasPrefix(dst, "/") {
		return errors.New("move: -destino inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("move: requiere sesión (login)")
	}

	if err := ext2.MoveNode(reg, s.ID, src, dst, s.UID, s.GID, s.IsRoot); err != nil {
		return err
	}

	_ = ext3.AppendJournalIfExt3(reg, s.ID, "MOVE", src, "dest="+dst)

	return nil
}
