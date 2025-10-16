package commands

import (
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/auth"
)

func CmdLogout(argv []string) int {
	_, ok := auth.Current()
	if !ok {
		fmt.Println("logout: no hay sesión activa")
		return 0
	}
	auth.Logout()
	fmt.Println("Sesión cerrada.")
	return 0
}
