package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Chown(reg *mount.Registry, path, newUser string, recursive bool) error {
	path = strings.TrimSpace(path)
	newUser = strings.TrimSpace(newUser)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("chown: -path inválido (debe ser absoluto)")
	}
	if newUser == "" {
		return errors.New("chown: -usuario requerido")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("chown: requiere sesión (login)")
	}

	// Aplica el cambio de propietario (root: cualquiera; no-root: solo propios)
	if err := ext2.Chown(reg, s.ID, path, newUser, recursive, s.UID, s.GID, s.IsRoot); err != nil {
		return err
	}

	// Journal (solo EXT3; no interrumpe si falla)
	_ = ext3.AppendJournalIfExt3(reg, s.ID, "CHOWN", path, fmt.Sprintf("usuario=%s recursive=%t", newUser, recursive))

	return nil
}
