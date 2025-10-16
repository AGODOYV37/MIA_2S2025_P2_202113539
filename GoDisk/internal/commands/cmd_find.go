package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdFind(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	path := fs.String("path", "", "Ruta de inicio (absoluta)")
	name := fs.String("name", "", "Patrón con ? (1) y * (1+)")
	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" || strings.TrimSpace(*name) == "" {
		fmt.Println("uso: find -path=/ruta -name=<patrón>")
		return 2
	}

	items, err := usersvc.Find(reg, *path, *name)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Println("(sin coincidencias)")
		return 0
	}
	for _, p := range items {
		fmt.Println(p)
	}
	return 0
}
