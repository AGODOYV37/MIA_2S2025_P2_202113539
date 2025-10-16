package mount

import (
	"sort"
	"sync"
)

type MountedPartition struct {
	DiskPath string
	PartName string
	Letter   rune
	Number   int
	ID       string
	Start    int64
	Size     int64
}

type MountedDisk struct {
	DiskPath string
	Letter   rune
	NextNum  int

	byName map[string]*MountedPartition
	byNum  map[int]*MountedPartition
}

type Registry struct {
	mu    sync.RWMutex
	disks map[string]*MountedDisk
	used  map[rune]bool
}

func NewRegistry() *Registry {
	return &Registry{
		disks: make(map[string]*MountedDisk),
		used:  make(map[rune]bool),
	}
}

func (r *Registry) IsMounted(diskPath, partName string) (*MountedPartition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	md, ok := r.disks[diskPath]
	if !ok {
		return nil, false
	}
	mp, ok := md.byName[partName]
	return mp, ok
}

func (r *Registry) PurgeDisk(diskPath string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	md, ok := r.disks[diskPath]
	if !ok {
		return 0
	}
	removed := 0

	for n, mp := range md.byNum {
		if mp == nil {
			continue
		}
		delete(md.byNum, n)
		delete(md.byName, mp.PartName)
		removed++
	}

	delete(r.used, md.Letter)

	delete(r.disks, diskPath)
	return removed
}

func (r *Registry) AddMountedPartition(mp *MountedPartition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	md, ok := r.disks[mp.DiskPath]
	if !ok {

		md = &MountedDisk{
			DiskPath: mp.DiskPath,
			Letter:   mp.Letter,
			NextNum:  mp.Number + 1,
			byName:   make(map[string]*MountedPartition),
			byNum:    make(map[int]*MountedPartition),
		}
		r.disks[mp.DiskPath] = md

		r.used[mp.Letter] = true
	}

	if _, exists := md.byName[mp.PartName]; exists {
		return ErrAlreadyMounted
	}

	if _, exists := md.byNum[mp.Number]; exists {
		return ErrAlreadyMounted
	}

	md.byName[mp.PartName] = mp
	md.byNum[mp.Number] = mp

	if mp.Number >= md.NextNum {
		md.NextNum = mp.Number + 1
	}
	return nil
}

func (r *Registry) GetByID(id string) (*MountedPartition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, md := range r.disks {
		for _, mp := range md.byNum {
			if mp != nil && mp.ID == id {
				return mp, true
			}
		}
	}
	return nil, false
}

func (r *Registry) ListIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type item struct {
		letter rune
		number int
		id     string
	}
	var items []item
	for _, md := range r.disks {
		for n, mp := range md.byNum {
			if mp == nil {
				continue
			}
			items = append(items, item{letter: md.Letter, number: n, id: mp.ID})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].letter == items[j].letter {
			return items[i].number < items[j].number
		}
		return items[i].letter < items[j].letter
	})
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.id)
	}
	return ids
}

func (r *Registry) RemoveByID(id string) (*MountedPartition, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for diskPath, md := range r.disks {
		for n, mp := range md.byNum {
			if mp != nil && mp.ID == id {

				delete(md.byNum, n)
				delete(md.byName, mp.PartName)

				_ = diskPath
				return mp, true
			}
		}
	}
	return nil, false
}

func (r *Registry) MountedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := 0
	for _, md := range r.disks {
		total += len(md.byName)
	}
	return total
}
