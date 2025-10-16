package mount

import (
	"fmt"
)

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

const carnetPrefix = "39"

func BuildID(number int, letter rune) string {
	return fmt.Sprintf("%s%d%c", carnetPrefix, number, letter)
}

func (r *Registry) letterForDisk(diskPath string) (rune, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	md, ok := r.disks[diskPath]
	if ok {
		return md.Letter, nil
	}

	for _, L := range letters {
		if !r.used[L] {
			r.used[L] = true
			r.disks[diskPath] = &MountedDisk{
				DiskPath: diskPath,
				Letter:   L,
				NextNum:  1,
				byName:   make(map[string]*MountedPartition),
				byNum:    make(map[int]*MountedPartition),
			}
			return L, nil
		}
	}
	return 0, ErrDiskLetterExhausted
}

func (r *Registry) nextNumForDisk(diskPath string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	md, ok := r.disks[diskPath]
	if !ok {

		L, err := r.assignDiskNoLock(diskPath)
		if err != nil {
			_ = L
			return 0, err
		}
		md = r.disks[diskPath]
	}
	n := md.NextNum
	if n <= 0 {
		n = 1
	}
	md.NextNum = n + 1
	return n, nil
}

func (r *Registry) assignDiskNoLock(diskPath string) (rune, error) {
	for _, L := range letters {
		if !r.used[L] {
			r.used[L] = true
			r.disks[diskPath] = &MountedDisk{
				DiskPath: diskPath,
				Letter:   L,
				NextNum:  1,
				byName:   make(map[string]*MountedPartition),
				byNum:    make(map[int]*MountedPartition),
			}
			return L, nil
		}
	}
	return 0, ErrDiskLetterExhausted
}
