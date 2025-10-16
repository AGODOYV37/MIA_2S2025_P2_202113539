package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
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

	// Requiere sesión (cualquier usuario). Root bypasses permisos en ext2.* helpers.
	s, err := auth.Require()
	if err != nil {
		return errors.New("move: requiere sesión (login)")
	}

	// La verificación de permisos (escritura sobre el ORIGEN) está en ext2.MoveNode
	return ext2.MoveNode(reg, s.ID, src, dst, s.UID, s.GID, s.IsRoot)
}
