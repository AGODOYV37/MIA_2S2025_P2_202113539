package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdEdit(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("edit", flag.ContinueOnError)
	path := cmd.String("path", "", "Ruta absoluta en EXT2/EXT3 (ej. /docs/nota.txt)")
	cont := cmd.String("cont", "", "Texto literal o ruta de archivo del SO")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Println("uso: edit -path=/ruta/archivo [-cont=\"texto\"|/ruta/host]")
		return 2
	}
	if err := usersvc.Edit(reg, *path, *cont); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("edit: actualizado %s\n", *path)
	return 0
}
