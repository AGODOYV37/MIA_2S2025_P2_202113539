package commands

import (
	"flag"
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func CmdMkfs(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkfs", flag.ExitOnError)
	id := cmd.String("id", "", "ID montado (generado por mount)")
	typ := cmd.String("type", "full", "Tipo de formateo (full)")
	cmd.Parse(argv)

	if *id == "" {
		fmt.Println("Error: -id es obligatorio.")
		return 2
	}
	_ = typ

	formatter := ext2.NewFormatter(reg)
	if err := formatter.MkfsFull(*id); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("mkfs: formateo EXT2 completado en", *id)
	return 0
}
