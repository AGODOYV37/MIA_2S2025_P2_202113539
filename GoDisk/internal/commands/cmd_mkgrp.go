package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdMkgrp(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkgrp", flag.ExitOnError)
	name := cmd.String("name", "", "Nombre del grupo (sin espacios ni comas)")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*name) == "" {
		fmt.Println("uso: mkgrp -name=<nombre>")
		return 2
	}
	if err := usersvc.Mkgrp(reg, *name); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("Grupo %q creado correctamente.\n", *name)
	return 0
}
