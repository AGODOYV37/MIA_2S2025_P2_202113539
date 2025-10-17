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

func Chmod(reg *mount.Registry, path, ugo string, recursive bool) error {
	path = strings.TrimSpace(path)
	ugo = strings.TrimSpace(ugo)

	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("chmod: -path inválido (debe ser absoluto)")
	}
	if len(ugo) != 3 {
		return errors.New("chmod: -ugo debe tener exactamente 3 dígitos (ej. 764)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("chmod: requiere sesión (login)")
	}
	// Solo root puede ejecutar chmod (según enunciado)
	if !s.IsRoot {
		return errors.New("chmod: operación permitida solo para root")
	}

	perms, err := ext2.ParseUGO(ugo)
	if err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	if err := ext2.Chmod(reg, s.ID, path, perms, recursive, s.UID, s.GID, s.IsRoot); err != nil {
		return err
	}

	// Journal (solo EXT3; no falla si el log no se puede escribir)
	_ = ext3.AppendJournalIfExt3(reg, s.ID, "CHMOD", path, fmt.Sprintf("ugo=%s recursive=%t", ugo, recursive))

	return nil
}
