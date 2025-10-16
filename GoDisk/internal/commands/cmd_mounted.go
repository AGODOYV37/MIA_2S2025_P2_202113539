package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

func CmdMounted(reg *mount.Registry, argv []string) int {
	flags := parseFlags(argv)

	if flags["-json"] {
		views, err := reg.MountedJSON()
		if err != nil {
			fmt.Println("(sin particiones montadas)")
			return 0
		}
		b, _ := json.MarshalIndent(views, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	if flags["-table"] {
		table, err := reg.MountedTable()
		if err != nil {
			fmt.Println("(sin particiones montadas)")
			return 0
		}
		fmt.Print(table)
		return 0
	}

	line, err := reg.MountedPlain()
	if err != nil {
		fmt.Println("(sin particiones montadas)")
		return 0
	}
	fmt.Println(line)
	return 0
}

func parseFlags(args []string) map[string]bool {
	out := map[string]bool{}
	for _, a := range args {
		a = strings.TrimSpace(strings.ToLower(a))

		switch a {
		case "-table", "--table":
			out["-table"] = true
		case "-json", "--json":
			out["-json"] = true
		}
	}
	return out
}
