package ext2

// Permisos básicos (r=4, w=2)
const (
	permR = 4
	permW = 2
)

func canRead(ino Inodo, uid, gid int, isRoot bool) bool {
	if isRoot {
		return true
	}
	switch {
	case int(ino.IUid) == uid:
		return (ino.IPerm[0] & permR) != 0
	case int(ino.IGid) == gid:
		return (ino.IPerm[1] & permR) != 0
	default:
		return (ino.IPerm[2] & permR) != 0
	}
}

func canWrite(ino Inodo, uid, gid int, isRoot bool) bool {
	if isRoot {
		return true
	}
	switch {
	case int(ino.IUid) == uid:
		return (ino.IPerm[0] & permW) != 0
	case int(ino.IGid) == gid:
		return (ino.IPerm[1] & permW) != 0
	default:
		return (ino.IPerm[2] & permW) != 0
	}
}
