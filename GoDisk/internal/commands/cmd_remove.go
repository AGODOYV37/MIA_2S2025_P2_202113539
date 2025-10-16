package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdRemove(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	path := fs.String("path", "", "Ruta absoluta a borrar dentro del FS (archivo o carpeta)")
	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" || !strings.HasPrefix(*path, "/") {
		fmt.Println("uso: remove -path=/ruta/absoluta")
		return 2
	}

	if err := usersvc.Remove(reg, *path); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("remove: eliminado %s\n", *path)
	return 0
}
