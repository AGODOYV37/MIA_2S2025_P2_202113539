package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/reports"
)

func CmdRep(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("rep", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "ID de la partición montada (p.ej. 391A)")
	name := fs.String("name", "", "Nombre del reporte (mbr, ...)")
	path := fs.String("path", "", "Ruta de salida (.json o .html)")
	ruta := fs.String("ruta", "", "Ruta interna opcional (según reporte)")

	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	params := reports.Params{
		ID:   strings.TrimSpace(*id),
		Name: reports.Name(strings.ToLower(strings.TrimSpace(*name))),
		Path: strings.TrimSpace(*path),
		Ruta: strings.TrimSpace(*ruta),
	}
	params.Clean()
	if err := params.Validate(); err != nil {
		fmt.Println("Error:", err)
		return 2
	}

	if err := reports.Generate(reg, params); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("rep: generado %s en %s\n", params.Name, params.Path)
	return 0
}
