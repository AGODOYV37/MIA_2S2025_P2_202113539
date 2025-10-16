package commands

import (
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func CmdMount(svc *mount.Service, argv []string) int {
	args := parseKV(argv)
	path := strings.TrimSpace(args["-path"])
	name := strings.TrimSpace(args["-name"])

	if path == "" || name == "" {
		fmt.Println("uso: mount -path=\"/ruta/al/disco.mia\" -name=\"NombreParticion\"")
		return 2
	}

	id, err := svc.Mount(path, name)
	if err != nil {

		switch {
		case mount.IsInvalidArgs(err):
			fmt.Println("Error: argumentos inválidos. uso: mount -path= -name=")
			return 2
		case mount.IsPartitionNotFound(err):
			fmt.Printf("Error: la partición %q no existe en el disco %s (o no es primaria).\n", name, path)
		case mount.IsNotPrimary(err):
			fmt.Printf("Error: la partición %q no es primaria (solo se montan primarias).\n", name)
		case mount.IsAlreadyMounted(err):
			fmt.Printf("Info: la partición %q ya estaba montada.\n", name)
		case mount.IsLetterExhausted(err):
			fmt.Println("Error: no hay letras disponibles para asignar a más discos (A..Z).")
		case mount.IsMBRRead(err):
			fmt.Printf("Error: no se pudo leer el MBR del disco: %s\n", path)
		case mount.IsMBRWrite(err):
			fmt.Printf("Error: no se pudo escribir el MBR del disco: %s\n", path)
		default:
			fmt.Println("Error:", err)
		}
		return 1
	}

	fmt.Printf("Montada %q en %s -> ID=%s\n", name, path, id)
	return 0
}

func parseKV(args []string) map[string]string {
	out := make(map[string]string, len(args))
	for _, a := range args {
		kv := strings.SplitN(a, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])

		val = strings.Trim(val, `"`)
		val = strings.Trim(val, `'`)
		out[key] = val
	}
	return out
}
