package usersvc

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func Chgrp(reg *mount.Registry, user, newGrp string) error {
	user = strings.TrimSpace(user)
	newGrp = strings.TrimSpace(newGrp)

	if user == "" || newGrp == "" {
		return errors.New("chgrp: faltan -user o -grp")
	}
	if strings.ContainsRune(user, ',') || strings.ContainsRune(newGrp, ',') {
		return errors.New("chgrp: user/grp no deben contener comas")
	}
	for _, r := range user {
		if unicode.IsSpace(r) {
			return errors.New("chgrp: user no debe contener espacios")
		}
	}
	for _, r := range newGrp {
		if unicode.IsSpace(r) {
			return errors.New("chgrp: grp no debe contener espacios")
		}
	}
	if strings.EqualFold(user, "root") {
		return errors.New("chgrp: no se permite cambiar el grupo del usuario root")
	}

	s, ok := auth.Current()
	if !ok {
		return errors.New("chgrp: no hay sesión activa; usa login")
	}
	if !s.IsRoot {
		return errors.New("chgrp: operación permitida solo para root")
	}

	txt, err := ext2.ReadUsersText(reg, s.ID)
	if err != nil {
		return fmt.Errorf("chgrp: %w", err)
	}

	groupExists := false
	lines := splitLines(txt)

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := splitCSV(line)
		if len(parts) < 3 {
			continue
		}
		if strings.EqualFold(parts[1], "G") {
			gid := atoiSafe(parts[0])
			gname := strings.TrimSpace(parts[2])
			if gid > 0 && gname == newGrp {
				groupExists = true
				break
			}
		}
	}
	if !groupExists {
		return fmt.Errorf("chgrp: el grupo %q no existe o está eliminado", newGrp)
	}

	found := false
	alreadySet := false

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := splitCSV(line)
		if len(parts) != 5 || !strings.EqualFold(parts[1], "U") {
			continue
		}
		uid := atoiSafe(parts[0])
		curGrp := strings.TrimSpace(parts[2])
		u := strings.TrimSpace(parts[3])
		pwd := strings.TrimSpace(parts[4])

		if u != user {
			continue
		}
		if uid == 0 {
			return fmt.Errorf("chgrp: el usuario %q ya está eliminado", user)
		}
		if curGrp == newGrp {
			alreadySet = true
			break
		}

		lines[i] = fmt.Sprintf("%d, U, %s, %s, %s", uid, newGrp, u, pwd)
		found = true
		break
	}

	if alreadySet {
		return fmt.Errorf("chgrp: el usuario %q ya pertenece al grupo %q", user, newGrp)
	}
	if !found {
		return fmt.Errorf("chgrp: el usuario %q no existe", user)
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	if err := ext2.RewriteUsers(reg, s.ID, newContent); err != nil {
		return fmt.Errorf("chgrp: no se pudo actualizar users.txt: %w", err)
	}
	return nil
}
