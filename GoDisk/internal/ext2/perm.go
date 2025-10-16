package ext2

func CanRead(ino Inodo, uid, gid int, isRoot bool) bool {
	if isRoot {
		return true
	}
	var p byte
	switch {
	case int(ino.IUid) == uid:
		p = ino.IPerm[0]
	case int(ino.IGid) == gid:
		p = ino.IPerm[1]
	default:
		p = ino.IPerm[2]
	}
	const R = 4
	return (int(p) & R) != 0
}

func CanWrite(ino Inodo, uid, gid int, isRoot bool) bool {
	if isRoot {
		return true
	}
	var p byte
	switch {
	case int(ino.IUid) == uid:
		p = ino.IPerm[0]
	case int(ino.IGid) == gid:
		p = ino.IPerm[1]
	default:
		p = ino.IPerm[2]
	}
	const W = 2
	return (int(p) & W) != 0
}
