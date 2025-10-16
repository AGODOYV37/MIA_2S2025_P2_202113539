package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Rmgrp(reg *mount.Registry, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, ',') {
		return errors.New("rmgrp: nombre inválido (no vacío ni con comas)")
	}

	s, ok := auth.Current()
	if !ok {
		return errors.New("rmgrp: no hay sesión activa; usa login")
	}
	if !s.IsRoot {
		return errors.New("rmgrp: operación permitida solo para root")
	}

	txt, err := ext2.ReadUsersText(reg, s.ID)
	if err != nil {
		return fmt.Errorf("rmgrp: %w", err)
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
		if tag != "G" {
			continue
		}
		gid := atoiSafe(parts[0])
		gname := strings.TrimSpace(parts[2])
		if gname != name {
			continue
		}

		if gid == 0 {
			alreadyDeleted = true
			break
		}

		lines[i] = fmt.Sprintf("0, G, %s", name)
		found = true
		break
	}

	if alreadyDeleted {
		return fmt.Errorf("rmgrp: el grupo %q ya está eliminado", name)
	}
	if !found {
		return fmt.Errorf("rmgrp: el grupo %q no existe", name)
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	if err := ext2.RewriteUsers(reg, s.ID, newContent); err != nil {
		return fmt.Errorf("rmgrp: no se pudo actualizar users.txt: %w", err)
	}
	return nil
}
