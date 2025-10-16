package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdCopy(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("copy", flag.ContinueOnError)
	src := cmd.String("path", "", "Ruta absoluta origen (archivo o carpeta). Ej: \"/docs/proy\"")
	dst := cmd.String("destino", "", "Ruta absoluta de CARPETA destino (debe existir). Ej: \"/backup\"")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*src) == "" || !strings.HasPrefix(*src, "/") {
		fmt.Println("uso: copy -path=\"/ruta/origen\" -destino=\"/ruta/carpeta_destino\"")
		return 2
	}
	if strings.TrimSpace(*dst) == "" || !strings.HasPrefix(*dst, "/") {
		fmt.Println("uso: copy -path=\"/ruta/origen\" -destino=\"/ruta/carpeta_destino\"")
		return 2
	}

	if err := usersvc.Copy(reg, *src, *dst); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("copy: completado (%s -> %s)\n", *src, *dst)
	return 0
}
