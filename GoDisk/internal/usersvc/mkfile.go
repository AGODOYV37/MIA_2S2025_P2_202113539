package usersvc

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Mkfile(reg *mount.Registry, path string, recursive bool, size int, cont string, force bool) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("mkfile: -path inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("mkfile: requiere sesión (login)")
	}

	if size < 0 {
		return errors.New("mkfile: -size no puede ser negativo")
	}

	var data []byte
	cont = strings.TrimSpace(cont)
	switch {
	case cont == "":

		if size > 0 {
			const pat = "0123456789"
			data = make([]byte, size)
			for i := 0; i < size; i++ {
				data[i] = pat[i%len(pat)]
			}
		} else {
			data = nil
		}

	case strings.HasPrefix(cont, "text:"):

		data = []byte(cont[len("text:"):])

	case strings.HasPrefix(cont, "file:"):

		host := strings.TrimSpace(cont[len("file:"):])
		b, err := os.ReadFile(host)
		if err != nil {
			return fmt.Errorf("mkfile: no pude leer -cont (file): %w", err)
		}
		data = b

	case strings.HasPrefix(cont, "@"):

		host := strings.TrimSpace(strings.TrimPrefix(cont, "@"))
		b, err := os.ReadFile(host)
		if err != nil {
			return fmt.Errorf("mkfile: no pude leer -cont (@file): %w", err)
		}
		data = b

	default:

		if fi, err := os.Stat(cont); err == nil && !fi.IsDir() {
			b, err := os.ReadFile(cont)
			if err != nil {
				return fmt.Errorf("mkfile: no pude leer -cont (file): %w", err)
			}
			data = b
		} else {
			data = []byte(cont) // literal
		}
	}

	if err := ext2.CreateOrOverwriteFile(reg, s.ID, path, data, recursive, force, s.UID, s.GID); err != nil {
		return err
	}

	_ = ext3.AppendJournalIfExt3(reg, s.ID, "MKFILE", path, string(data))

	return nil
}
