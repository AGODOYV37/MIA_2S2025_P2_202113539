package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/usersvc"
)

func CmdMkfile(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkfile", flag.ContinueOnError)

	path := cmd.String("path", "", "Ruta absoluta en EXT2 (ej. /docs/nota.txt)")
	recursive := cmd.Bool("r", false, "Crear padres si no existen")
	sizeU := cmd.Uint("size", 0, "Tamaño (bytes, no negativos) si no se usa -cont")
	cont := cmd.String("cont", "", "Ruta de archivo de texto en el SO; tiene prioridad sobre -size")
	force := cmd.Bool("force", false, "Sobrescribir si el archivo ya existe (sin preguntar)")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	if strings.TrimSpace(*path) == "" {
		fmt.Println("uso: mkfile -path=/ruta/archivo [-r] [-size=N] [-cont=/ruta/host] [-force]")
		return 2
	}

	size := int(*sizeU)

	if err := usersvc.Mkfile(reg, *path, *recursive, size, *cont, *force); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	fmt.Printf("mkfile: creado/actualizado %s\n", *path)
	return 0
}
