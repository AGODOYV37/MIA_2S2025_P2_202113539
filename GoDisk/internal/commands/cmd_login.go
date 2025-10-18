package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdLogin(reg *mount.Registry, argv []string) int {
	// NO usar ExitOnError porque hace os.Exit(2) y tumba el servidor en HTTP
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // evita que flag imprima a stderr (rompe la respuesta capturada)

	id := fs.String("id", "", "ID de partición montada (p. ej. 391A)")
	user := fs.String("user", "", "Usuario")
	pass := fs.String("pass", "", "Contraseña")

	// Aliases para compatibilidad con el front o scripts anteriores
	usrAlias := fs.String("usr", "", "alias de -user")
	pwdAlias := fs.String("pwd", "", "alias de -pass")

	if err := fs.Parse(argv); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	// Resuelve alias
	if strings.TrimSpace(*user) == "" && strings.TrimSpace(*usrAlias) != "" {
		*user = *usrAlias
	}
	if strings.TrimSpace(*pass) == "" && strings.TrimSpace(*pwdAlias) != "" {
		*pass = *pwdAlias
	}

	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*user) == "" || strings.TrimSpace(*pass) == "" {
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
