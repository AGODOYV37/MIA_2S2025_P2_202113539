package usersvc

import (
	"errors"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/auth"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

func JournalingJSON(reg *mount.Registry, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {

		if s, ok := auth.Current(); ok {
			id = s.ID
		} else {
			return "", errors.New("journaling: especifica -id o inicia sesión")
		}
	}
	return ext3.ListJournalJSON(reg, id)
}
