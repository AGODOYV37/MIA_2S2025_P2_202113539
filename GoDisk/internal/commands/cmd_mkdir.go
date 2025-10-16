package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/usersvc"
)

func CmdMkdir(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkdir", flag.ContinueOnError)
	path := cmd.String("path", "", "Ruta absoluta en EXT2 (ej. /docs/proyectos)")
	p := cmd.Bool("p", false, "Crear padres si no existen (mkdir -p)")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Println("uso: mkdir -path=/ruta [-p]")
		return 2
	}

	if err := usersvc.Mkdir(reg, *path, *p); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("mkdir: creada %s\n", *path)
	return 0
}
