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
		// sin -cont: usa -size si viene
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
		// Forzar literal
		data = []byte(cont[len("text:"):])

	case strings.HasPrefix(cont, "file:"):
		// Forzar ruta de host
		host := strings.TrimSpace(cont[len("file:"):])
		b, err := os.ReadFile(host)
		if err != nil {
			return fmt.Errorf("mkfile: no pude leer -cont (file): %w", err)
		}
		data = b

	case strings.HasPrefix(cont, "@"):
		// Convención tipo @/ruta/host para ruta explícita
		host := strings.TrimSpace(strings.TrimPrefix(cont, "@"))
		b, err := os.ReadFile(host)
		if err != nil {
			return fmt.Errorf("mkfile: no pude leer -cont (@file): %w", err)
		}
		data = b

	default:
		// Heurística: si existe como archivo en el host -> léelo; si no, trátalo como texto literal.
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

	// Crear / sobrescribir archivo
	if err := ext2.CreateOrOverwriteFile(reg, s.ID, path, data, recursive, force, s.UID, s.GID); err != nil {
		return err
	}

	// Registrar en journal si la partición es EXT3
	_ = ext3.AppendJournalIfExt3(reg, s.ID, "MKFILE", path, string(data))

	return nil
}
