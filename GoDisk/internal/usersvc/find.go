package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Find(reg *mount.Registry, startPath, namePattern string) ([]string, error) {
	startPath = strings.TrimSpace(startPath)
	if startPath == "" || !strings.HasPrefix(startPath, "/") {
		return nil, errors.New("find: -path inválido (debe ser absoluto)")
	}
	if strings.TrimSpace(namePattern) == "" {
		namePattern = "*"
	}

	s, err := auth.Require()
	if err != nil {
		return nil, errors.New("find: requiere sesión (login)")
	}

	return ext2.Find(reg, s.ID, startPath, namePattern, s.UID, s.GID, s.IsRoot)
}
