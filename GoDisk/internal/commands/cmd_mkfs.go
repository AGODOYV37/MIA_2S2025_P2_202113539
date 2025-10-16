package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdMkfs(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkfs", flag.ExitOnError)
	id := cmd.String("id", "", "ID montado (generado por mount)")
	typ := cmd.String("type", "full", "Tipo de formateo (full)")
	fstype := cmd.String("fs", "ext2", "Sistema de archivos: ext2|ext3")
	cmd.Parse(argv)

	if *id == "" {
		fmt.Println("Error: -id es obligatorio.")
		return 2
	}
	if strings.ToLower(*typ) != "full" {
		fmt.Println("Aviso: solo -type=full.")
	}

	switch strings.ToLower(*fstype) {
	case "ext3":
		if err := ext3.NewFormatter(reg).MkfsFull(*id); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("mkfs: formateo EXT3 completado en", *id)
	default:
		if err := ext2.NewFormatter(reg).MkfsFull(*id); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("mkfs: formateo EXT2 completado en", *id)
	}
	return 0
}
