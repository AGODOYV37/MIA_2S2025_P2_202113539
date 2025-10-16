package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

type Session struct {
	ID     string
	User   string
	Group  string
	UID    int
	GID    int
	IsRoot bool
}

var (
	mu      sync.RWMutex
	current *Session
)

func Current() (*Session, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if current == nil {
		return nil, false
	}
	c := *current
	return &c, true
}

func Logout() {
	mu.Lock()
	defer mu.Unlock()
	current = nil
}

func Login(reg *mount.Registry, id, user, pass string) error {
	user = strings.TrimSpace(user)
	pass = strings.TrimSpace(pass)
	if user == "" || pass == "" || strings.TrimSpace(id) == "" {
		return errors.New("login: parámetros inválidos (-id, -user, -pass)")
	}

	mu.Lock()
	if current != nil {
		mu.Unlock()
		return errors.New("login: ya hay una sesión activa; usa logout primero")
	}
	mu.Unlock()

	txt, err := ext2.ReadUsersText(reg, id)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	groups := map[string]int{}
	type userRec struct {
		uid         int
		group, pass string
		active      bool
	}
	users := map[string]userRec{}

	for _, line := range splitLines(txt) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := splitCSV(line)
		if len(parts) < 3 {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(parts[1]))
		switch tag {
		case "G":
			if len(parts) != 3 {
				continue
			}
			gid := atoiSafe(parts[0])
			name := strings.TrimSpace(parts[2])
			if gid == 0 || name == "" {
				continue
			}
			groups[name] = gid
		case "U":
			if len(parts) != 5 {
				continue
			}
			uid := atoiSafe(parts[0])
			grp := strings.TrimSpace(parts[2])
			usr := strings.TrimSpace(parts[3])
			pwd := strings.TrimSpace(parts[4])
			users[usr] = userRec{uid: uid, group: grp, pass: pwd, active: uid != 0}
		}
	}

	u, ok := users[user]
	if !ok || !u.active {
		return errors.New("login: usuario no existe o está eliminado")
	}
	if u.pass != pass {
		return errors.New("login: contraseña incorrecta")
	}
	gid, gexists := groups[u.group]
	if !gexists {
		return errors.New("login: el grupo del usuario no existe o está eliminado")
	}

	sess := &Session{
		ID:     id,
		User:   user,
		Group:  u.group,
		UID:    u.uid,
		GID:    gid,
		IsRoot: user == "root",
	}
	mu.Lock()
	current = sess
	mu.Unlock()
	return nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		} else {
			break
		}
	}
	return n
}

func Require() (*Session, error) {
	s, ok := Current()
	if !ok {
		return nil, errors.New("requiere sesión (login)")
	}
	return s, nil
}
