package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdLogin(reg *mount.Registry, argv []string) int {
	cmd := flag.NewFlagSet("login", flag.ExitOnError)
	id := cmd.String("id", "", "ID de partición montada (p. ej. 391A)")
	user := cmd.String("user", "", "Usuario")
	pass := cmd.String("pass", "", "Contraseña")
	if err := cmd.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if strings.TrimSpace(*id) == "" || *user == "" || *pass == "" {
		fmt.Println("uso: login -id=<ID> -user=<usuario> -pass=<contraseña>")
		return 2
	}
	if err := auth.Login(reg, *id, *user, *pass); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if s, ok := auth.Current(); ok {
		rol := "usuario"
		if s.IsRoot {
			rol = "root"
		}
		fmt.Printf("Sesión iniciada: %s (uid=%d, gid=%d) en %s\n", s.User, s.UID, s.GID, s.ID)
		fmt.Printf("Rol: %s, grupo: %s\n", rol, s.Group)
	}
	return 0
}
