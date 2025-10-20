package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdJournaling(reg *mount.Registry, argv []string) int {
	fs := flag.NewFlagSet("journaling", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "ID de partición montada (opcional; si omites y tienes sesión activa, se usa esa partición)")

	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	jsonStr, err := usersvc.JournalingJSON(reg, strings.TrimSpace(*id))
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	fmt.Println(jsonStr)
	return 0
}
