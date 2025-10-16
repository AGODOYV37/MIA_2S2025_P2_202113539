package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func Rmusr(reg *mount.Registry, user string) error {
	user = strings.TrimSpace(user)
	if user == "" || invalidToken(user) {
		return errors.New("rmusr: -usr inválido (no vacío, sin espacios ni comas)")
	}
	if strings.EqualFold(user, "root") {
		return errors.New("rmusr: no se permite eliminar al usuario root")
	}

	s, ok := auth.Current()
	if !ok {
		return errors.New("rmusr: no hay sesión activa; usa login")
	}
	if !s.IsRoot {
		return errors.New("rmusr: operación permitida solo para root")
	}

	txt, err := ext2.ReadUsersText(reg, s.ID)
	if err != nil {
		return fmt.Errorf("rmusr: %w", err)
	}

	lines := splitLines(txt)
	found := false
	alreadyDeleted := false

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := splitCSV(line)
		if len(parts) < 3 {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(parts[1]))
		if tag != "U" || len(parts) != 5 {
			continue
		}
		uid := atoiSafe(parts[0])
		grp := strings.TrimSpace(parts[2])
		u := strings.TrimSpace(parts[3])
		pwd := strings.TrimSpace(parts[4])

		if u != user {
			continue
		}
		if uid == 0 {
			alreadyDeleted = true
			break
		}

		lines[i] = fmt.Sprintf("0, U, %s, %s, %s", grp, u, pwd)
		found = true
		break
	}

	if alreadyDeleted {
		return fmt.Errorf("rmusr: el usuario %q ya está eliminado", user)
	}
	if !found {
		return fmt.Errorf("rmusr: el usuario %q no existe", user)
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	if err := ext2.RewriteUsers(reg, s.ID, newContent); err != nil {
		return fmt.Errorf("rmusr: no se pudo actualizar users.txt: %w", err)
	}
	return nil
}
