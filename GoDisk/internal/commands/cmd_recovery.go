package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdRecovery(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("recovery", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "ID montado (generado por mount)")

	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Println("uso: recovery -id=<ID>")
		return 2
	}
	if err := ext3.Recover(reg, *id); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("recovery: completado")
	return 0
}
