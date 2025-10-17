package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdLoss(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("loss", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "ID montado (p. ej. 061Disco1)")

	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Println("uso: loss -id=<ID>")
		return 2
	}

	if err := ext3.Loss(reg, *id); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	fmt.Printf("loss: aplicado en %s (bitmap de inodos, bitmap de bloques, inodos y bloques limpiados)\n", *id)
	return 0
}
