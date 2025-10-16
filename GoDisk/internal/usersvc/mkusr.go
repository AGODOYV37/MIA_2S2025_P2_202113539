package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Mkusr(reg *mount.Registry, user, pass, grp string) error {
	user = strings.TrimSpace(user)
	pass = strings.TrimSpace(pass)
	grp = strings.TrimSpace(grp)

	if user == "" || pass == "" || grp == "" {
		return errors.New("mkusr: faltan -usr, -pass o -grp")
	}
	if invalidToken(user) || invalidToken(pass) || invalidToken(grp) {
		return errors.New("mkusr: usr/pass/grp no deben contener espacios ni comas")
	}

	s, err := auth.Require()
	if err != nil {
		return errors.New("mkusr: requiere sesión (login)")
	}
	if !s.IsRoot {
		return errors.New("mkusr: operación permitida solo para root")
	}

	txt, err := ext2.ReadUsersText(reg, s.ID)
	if err != nil {
		return fmt.Errorf("mkusr: %w", err)
	}

	groupExists := false
	maxUID := 0

	for _, line := range splitLines(txt) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := splitCSV(line)
		if len(parts) < 3 {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(parts[1]))

		switch tag {
		case "G":
			if len(parts) != 3 {
				continue
			}
			gid := atoiSafe(parts[0])
			gname := strings.TrimSpace(parts[2])
			if gid > 0 && gname == grp {
				groupExists = true
			}
		case "U":
			if len(parts) != 5 {
				continue
			}
			uid := atoiSafe(parts[0])
			uuser := strings.TrimSpace(parts[3])
			if uid > 0 && uuser == user {
				return fmt.Errorf("mkusr: el usuario %q ya existe", user)
			}
			if uid > maxUID {
				maxUID = uid
			}
		}
	}

	if !groupExists {
		return fmt.Errorf("mkusr: el grupo %q no existe o está eliminado", grp)
	}

	newUID := maxUID + 1
	line := fmt.Sprintf("%d, U, %s, %s, %s", newUID, grp, user, pass)

	if err := ext2.AppendUsersLine(reg, s.ID, line); err != nil {
		return fmt.Errorf("mkusr: no se pudo escribir users.txt: %w", err)
	}
	return nil
}
