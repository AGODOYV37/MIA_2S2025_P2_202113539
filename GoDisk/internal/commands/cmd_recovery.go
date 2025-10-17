package commands

import (
	"flag"
	"fmt"
	"io"
	"sort"
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

	rep, err := ext3.RecoverWithReport(reg, *id)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	fmt.Println("recovery: completado")
	fmt.Printf("Procesadas: %d | aplicadas: %d | omitidas: %d | errores: %d\n",
		rep.Total, rep.Applied, rep.Skipped, rep.Failed)

	// Imprimir ByOp ordenado por clave
	if len(rep.ByOp) > 0 {
		keys := make([]string, 0, len(rep.ByOp))
		for k := range rep.ByOp {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Print("Por operación: ")
		for i, k := range keys {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s=%d", k, rep.ByOp[k])
		}
		fmt.Println()
	}

	if len(rep.Details) > 0 {
		fmt.Println("Detalles:")
		for _, d := range rep.Details {
			fmt.Println("-", d)
		}
	}

	return 0
}
