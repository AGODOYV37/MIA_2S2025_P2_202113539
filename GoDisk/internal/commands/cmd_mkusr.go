package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/usersvc"
)

func CmdMkusr(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("mkusr", flag.ContinueOnError)
	cmd.SetOutput(io.Discard)
	user := cmd.String("user", "", "Usuario (sin espacios ni comas)")
	pass := cmd.String("pass", "", "Contraseña (sin espacios ni comas)")
	grp := cmd.String("grp", "", "Grupo existente (activo)")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*user) == "" || strings.TrimSpace(*pass) == "" || strings.TrimSpace(*grp) == "" {
		fmt.Println("uso: mkusr -usr=<usuario> -pass=<contraseña> -grp=<grupo>")
		return 2
	}

	if err := usersvc.Mkusr(reg, *user, *pass, *grp); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Printf("Usuario %q creado en grupo %q.\n", *user, *grp)
	return 0
}
