package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/usersvc"
)

func CmdRmgrp(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("rmgrp", flag.ExitOnError)
	name := cmd.String("name", "", "Nombre del grupo a eliminar (borrado lógico)")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*name) == "" {
		fmt.Println("uso: rmgrp -name=<grupo>")
		return 2
	}

	if err := usersvc.Rmgrp(reg, *name); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("Grupo %q eliminado lógicamente (gid=0).\n", *name)
	return 0
}
