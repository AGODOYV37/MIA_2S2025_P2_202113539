package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Cat(reg *mount.Registry, files []string) (string, error) {
	if len(files) == 0 {
		return "", errors.New("cat: especifica al menos un -fileN")
	}
	s, err := auth.Require()
	if err != nil {
		return "", errors.New("cat: requiere sesión (login)")
	}

	var b strings.Builder
	for i, p := range files {
		p = strings.TrimSpace(p)
		if p == "" || !strings.HasPrefix(p, "/") {
			return "", fmt.Errorf("cat: ruta inválida: %q", p)
		}

		ino, data, err := ext2.ReadFileByPath(reg, s.ID, p)
		if err != nil {
			return "", err
		}
		if !ext2.CanRead(ino, s.UID, s.GID, s.IsRoot) {
			return "", fmt.Errorf("cat: permiso denegado para %s", p)
		}

		if i > 0 {
			b.WriteByte('\n')
		}
		b.Write(data)
	}
	return b.String(), nil
}
