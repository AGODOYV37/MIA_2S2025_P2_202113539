package ext2

var (
	ErrPartTooSmall = fmtError("ext2: partición demasiado pequeña")
)

type fmtError string

func (e fmtError) Error() string { return string(e) }

func NewBitmaps(nIn, nBl int32) ([]byte, []byte) {
	return make([]byte, nIn), make([]byte, nBl)
}
func MarkInode(bm []byte, idx int32, used bool) {
	if used {
		bm[idx] = 1
	} else {
		bm[idx] = 0
	}
}
func MarkBlock(bm []byte, idx int32, used bool) {
	if used {
		bm[idx] = 1
	} else {
		bm[idx] = 0
	}
}
func FirstFree(bm []byte) int32 {
	for i := range bm {
		if bm[i] == 0 {
			return int32(i)
		}
	}
	return -1
}
