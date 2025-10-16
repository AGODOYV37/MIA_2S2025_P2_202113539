package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdChgrp(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("chgrp", flag.ContinueOnError)
	cmd.SetOutput(io.Discard)

	user := cmd.String("user", "", "Usuario existente (activo)")
	grp := cmd.String("grp", "", "Nuevo grupo existente (activo)")

	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*user) == "" || strings.TrimSpace(*grp) == "" {
		fmt.Println("uso: chgrp -user=<usuario> -grp=<grupo>")
		return 2
	}

	if err := usersvc.Chgrp(reg, *user, *grp); err != nil {
		// No detiene el proceso, solo reporta
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("Grupo del usuario %q actualizado a %q.\n", *user, *grp)
	return 0
}
