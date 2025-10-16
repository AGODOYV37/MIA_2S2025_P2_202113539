package commands

import (
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/usersvc"
)

func CmdCat(reg *mount.Registry, argv []string) int {
	files, err := parseFileArgs(argv)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	out, err := usersvc.Cat(reg, files)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if out != "" {
		fmt.Println(out)
	}
	return 0
}

func parseFileArgs(argv []string) ([]string, error) {
	files := []string{}
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-file") {
			continue
		}
		// -fileN=/path
		if eq := strings.IndexByte(arg, '='); eq >= 0 {
			val := strings.Trim(arg[eq+1:], `"`)
			if val == "" {
				return nil, fmt.Errorf("cat: parámetro %q sin valor", arg[:eq])
			}
			files = append(files, val)
			continue
		}
		// -fileN /path
		if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
			i++
			val := strings.Trim(argv[i], `"`)
			if val == "" {
				return nil, fmt.Errorf("cat: parámetro %q sin valor", arg)
			}
			files = append(files, val)
			continue
		}
		return nil, fmt.Errorf("cat: parámetro %q requiere un valor", arg)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("uso: cat -file1=/ruta [/más -fileN=/ruta]")
	}
	return files, nil
}
