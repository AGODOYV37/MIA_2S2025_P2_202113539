package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdRename(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("rename", flag.ContinueOnError)
	path := cmd.String("path", "", "Ruta absoluta del archivo/carpeta (ej. /docs/nota.txt)")
	name := cmd.String("name", "", "Nuevo nombre (≤12, sin espacios/comas)")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" || strings.TrimSpace(*name) == "" {
		fmt.Println("uso: rename -path=/ruta/actual -name=NuevoNombre")
		return 2
	}

	if err := usersvc.Rename(reg, *path, *name); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("rename: '%s' ahora se llama '%s'\n", *path, *name)
	return 0
}
