package usersvc

import (
	"fmt"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

type sessionInfo struct {
	ID  string
	UID int
	GID int
}

// NOTE: Reemplaza esta función por la que YA USAS internamente para Mkdir/Mkfile.
// Debe devolver el id de partición montada y el uid/gid del usuario logueado.
func requireLogin() (sessionInfo, error) {
	// EJEMPLO: si ya tienes algo como Current():
	// s := Current()
	// if !s.Logged { return sessionInfo{}, fmt.Errorf("no hay sesión activa") }
	// return sessionInfo{ID: s.MountedID, UID: s.UID, GID: s.GID}, nil
	return sessionInfo{}, fmt.Errorf("usersvc: implementa requireLogin() según tu sesión")
}

func Remove(reg *mount.Registry, absPath string) error {
	s, err := requireLogin()
	if err != nil {
		return err
	}
	return ext2.Remove(reg, s.ID, absPath, s.UID, s.GID)
}
