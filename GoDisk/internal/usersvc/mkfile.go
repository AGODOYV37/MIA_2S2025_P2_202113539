package usersvc

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Mkfile(reg *mount.Registry, path string, recursive bool, size int, contPath string, force bool) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("mkfile: -path inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("mkfile: requiere sesión (login)")
	}
	if !s.IsRoot {
		return errors.New("mkfile: operación permitida solo para root")
	}

	if size < 0 {
		return errors.New("mkfile: -size no puede ser negativo")
	}

	var data []byte
	if strings.TrimSpace(contPath) != "" {
		b, err := os.ReadFile(contPath)
		if err != nil {
			return fmt.Errorf("mkfile: no pude leer -cont: %w", err)
		}
		data = b
	} else if size > 0 {
		const pat = "0123456789"
		data = make([]byte, size)
		for i := 0; i < size; i++ {
			data[i] = pat[i%len(pat)]
		}
	} else {
		data = nil
	}

	return ext2.CreateOrOverwriteFile(reg, s.ID, path, data, recursive, force, s.UID, s.GID)
}
