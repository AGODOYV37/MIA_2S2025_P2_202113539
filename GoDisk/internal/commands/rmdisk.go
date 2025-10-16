package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteRmdisk elimina el archivo .mia de forma no interactiva.
//   - Si el archivo no existe, lo toma como éxito (idempotente).
//   - Nunca bloquea ni termina el proceso; solo imprime y retorna un bool
//     que indica si se debe remover del catálogo.
func ExecuteRmdisk(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		fmt.Println("rmdisk: -path requerido")
		return false
	}
	ap := filepath.Clean(path)

	if err := os.Remove(ap); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("rmdisk: el archivo ya no existe (idempotente)")
			return true
		}
		fmt.Printf("rmdisk: no se pudo eliminar %q: %v\n", ap, err)
		return false
	}

	fmt.Println("rmdisk: eliminado", ap)
	return true
}
