package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdChown(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("chown", flag.ContinueOnError)
	path := cmd.String("path", "", "Ruta absoluta en EXT2 (archivo o carpeta)")
	user := cmd.String("usuario", "", "Nuevo propietario (usuario existente)")
	rec := cmd.Bool("r", false, "Aplicar recursivamente (si es carpeta)")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" || !strings.HasPrefix(*path, "/") {
		fmt.Println("chown: -path inválido (debe ser absoluto)")
		return 2
	}
	if strings.TrimSpace(*user) == "" {
		fmt.Println("chown: -usuario requerido")
		return 2
	}

	if err := usersvc.Chown(reg, *path, *user, *rec); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("chown: propietario actualizado")
	return 0
}
