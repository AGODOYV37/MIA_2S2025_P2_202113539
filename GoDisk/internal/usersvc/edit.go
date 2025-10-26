package usersvc

import (
	"errors"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

// Edit ahora acepta `cont` que puede ser texto literal o ruta a un archivo del SO.
func Edit(reg *mount.Registry, path string, cont string) error {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("edit: -path inválido (debe ser absoluto)")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("edit: requiere sesión (login)")
	}

	// Resolver contenido: si cont apunta a un archivo existente en el SO, se usa;
	// en caso contrario, se trata como texto literal (tal cual).
	data, err := resolveEditContent(cont)
	if err != nil {
		return err
	}

	// Editar el archivo (requiere rw o root, validado en ext2.EditFile)
	if err := ext2.EditFile(reg, s.ID, path, data, s.UID, s.GID, s.IsRoot); err != nil {
		return err
	}

	// Journaling (EXT3): guardamos el texto final aplicado
	_ = ext3.AppendJournalIfExt3(reg, s.ID, "EDIT", path, string(data))
	return nil
}

func resolveEditContent(cont string) ([]byte, error) {
	cont = strings.TrimSpace(cont)
	if cont == "" {
		// Permite vaciar el archivo
		return []byte{}, nil
	}
	// Intentar leerlo como ruta de archivo del SO
	if b, err := os.ReadFile(cont); err == nil {
		return b, nil
	}
	// Si no existe como archivo, usarlo como texto literal
	return []byte(cont), nil
}
