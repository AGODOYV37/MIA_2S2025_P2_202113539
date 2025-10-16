package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func Mkgrp(reg *mount.Registry, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t\r\n") || strings.ContainsRune(name, ',') {
		return errors.New("mkgrp: nombre inválido (no puede estar vacío ni contener espacios o comas)")
	}

	s, ok := auth.Current()
	if !ok {
		return errors.New("mkgrp: no hay sesión activa; usa login")
	}
	if !s.IsRoot {
		return errors.New("mkgrp: operación permitida solo para root")
	}

	txt, err := ext2.ReadUsersText(reg, s.ID)
	if err != nil {
		return fmt.Errorf("mkgrp: %w", err)
	}

	maxGID := 0
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
		if tag != "G" {
			continue
		}
		gid := atoiSafe(parts[0])
		gname := strings.TrimSpace(parts[2])

		if gid > 0 && gname == name {
			return fmt.Errorf("mkgrp: el grupo %q ya existe", name)
		}
		if gid > maxGID {
			maxGID = gid
		}
	}

	newGID := maxGID + 1
	line := fmt.Sprintf("%d, G, %s", newGID, name)
	if err := ext2.AppendUsersLine(reg, s.ID, line); err != nil {
		return fmt.Errorf("mkgrp: no se pudo escribir users.txt: %w", err)
	}
	return nil
}
