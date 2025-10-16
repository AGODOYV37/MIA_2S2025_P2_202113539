package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdMove(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("move", flag.ContinueOnError)
	src := cmd.String("path", "", "Ruta absoluta del archivo/carpeta origen")
	dst := cmd.String("destino", "", "Ruta absoluta de la carpeta destino")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*src) == "" || strings.TrimSpace(*dst) == "" {
		fmt.Println("uso: move -path=/origen -destino=/carpeta_destino")
		return 2
	}
	if err := usersvc.Move(reg, *src, *dst); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("move: '%s' -> '%s' (OK)\n", *src, *dst)
	return 0
}
