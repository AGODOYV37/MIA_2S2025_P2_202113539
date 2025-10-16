package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdEdit(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("edit", flag.ContinueOnError)

	path := cmd.String("path", "", "Ruta absoluta del archivo en EXT2/3 (ej. /docs/nota.txt)")
	sizeU := cmd.Uint("size", 0, "Tamaño en bytes del nuevo contenido (genera patrón 012345...)")
	cont := cmd.String("cont", "", "Ruta de archivo de texto en el SO; tiene prioridad sobre -size")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Println("uso: edit -path=/ruta/archivo [-cont=/ruta/host] [-size=N]")
		return 2
	}
	if *cont != "" {
		b, err := os.ReadFile(*cont)
		if err != nil {
			fmt.Printf("Error: no pude leer -cont: %v\n", err)
			return 1
		}
		if err := usersvc.Edit(reg, *path, b); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Printf("edit: contenido actualizado en %s (desde -cont)\n", *path)
		return 0
	}

	// Genera patrón por tamaño
	size := int(*sizeU)
	if size < 0 {
		fmt.Println("Error: -size no puede ser negativo")
		return 2
	}
	var data []byte
	if size > 0 {
		const pat = "0123456789"
		data = make([]byte, size)
		for i := 0; i < size; i++ {
			data[i] = pat[i%len(pat)]
		}
	} else {
		// size==0 => archivo queda vacío
		data = nil
	}
	if err := usersvc.Edit(reg, *path, data); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("edit: contenido actualizado en %s\n", *path)
	return 0
}
