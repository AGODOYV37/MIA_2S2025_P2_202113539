package mount

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/catalog"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/diskio"
	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/structs"
)

func (r *Registry) RehydrateFromCatalog() error {
	paths, err := catalog.All()
	if err != nil {
		return fmt.Errorf("rehydrate: leyendo catálogo: %w", err)
	}
	if len(paths) == 0 {
		return nil
	}
	return r.RehydrateFromDisks(paths)
}

func (r *Registry) RehydrateFromDisks(diskPaths []string) error {
	for _, raw := range diskPaths {
		path := filepath.Clean(raw)

		dir := filepath.Dir(path)
		if !dirExists(dir) {
			fmt.Printf("rehydrate: la carpeta %q no existe; removiendo del catálogo\n", dir)
			_ = catalog.Remove(path)
			continue
		}
		if !fileExists(path) {
			fmt.Printf("rehydrate: el archivo %q no existe; removiendo del catálogo\n", path)
			_ = catalog.Remove(path)
			continue
		}

		mbr, err := diskio.ReadMBR(path)
		if err != nil {

			if os.IsNotExist(err) {
				fmt.Printf("rehydrate: %q ya no existe; removiendo del catálogo\n", path)
				_ = catalog.Remove(path)
				continue
			}

			fmt.Printf("rehydrate: no se pudo leer MBR de %q: %v\n", path, err)
			continue
		}

		type mountInfo struct {
			name      string
			id        string
			number    int
			start     int64
			size      int64
			idLetter  rune
			hasID     bool
			hasNumber bool
		}
		var mounts []mountInfo
		var maxCorrelative int
		var foundLetter rune

		for i := range mbr.Mbr_partitions {
			p := &mbr.Mbr_partitions[i]
			if !isPrimaryUsable(p) || p.Part_status != '1' {
				continue
			}
			name := bytesToString(p.Part_name[:])
			id := bytesToString(p.Part_id[:])

			var (
				num    = int(p.Part_correlative)
				hasNum = num > 0
				letter rune
				hasID  bool
			)

			if id != "" && len(id) >= 3 {

				letter = rune(id[len(id)-1])
				hasID = true
				if letter != 0 && foundLetter == 0 {
					foundLetter = letter
				}
			}

			if !hasNum {
				if n, ok := parseNumberFromID(id); ok {
					num = n
					hasNum = true
				}
			}
			if hasNum && num > maxCorrelative {
				maxCorrelative = num
			}

			mounts = append(mounts, mountInfo{
				name:      name,
				id:        id,
				number:    num,
				start:     p.Part_start,
				size:      p.Part_s,
				idLetter:  letter,
				hasID:     hasID,
				hasNumber: hasNum,
			})
		}

		if len(mounts) == 0 {
			continue
		}

		var diskLetter rune
		if foundLetter != 0 {
			diskLetter = foundLetter
		} else {
			L, err := r.letterForDisk(path)
			if err != nil {
				diskLetter = 'A'
			} else {
				diskLetter = L
			}
		}

		r.mu.Lock()
		md, exists := r.disks[path]
		if !exists {
			md = &MountedDisk{
				DiskPath: path,
				Letter:   diskLetter,
				NextNum:  1,
				byName:   make(map[string]*MountedPartition),
				byNum:    make(map[int]*MountedPartition),
			}
			r.disks[path] = md
		} else if md.Letter == 0 {
			md.Letter = diskLetter
		}

		r.used[md.Letter] = true
		r.mu.Unlock()

		for _, mi := range mounts {

			id := mi.id
			if id == "" && mi.hasNumber {
				id = BuildID(mi.number, md.Letter)
			}

			num := mi.number
			if !mi.hasNumber {
				continue
			}

			mp := &MountedPartition{
				DiskPath: path,
				PartName: mi.name,
				Letter:   md.Letter,
				Number:   num,
				ID:       id,
				Start:    mi.start,
				Size:     mi.size,
			}
			_ = r.AddMountedPartition(mp)
		}

		r.mu.Lock()
		if md, ok := r.disks[path]; ok {
			if maxCorrelative+1 > md.NextNum {
				md.NextNum = maxCorrelative + 1
			}
		}
		r.mu.Unlock()
	}
	return nil
}

func isPrimaryUsable(p *structs.Partition) bool {
	return (p.Part_type == 'P' || p.Part_type == 'p') && p.Part_s > 0 && p.Part_start >= 0
}

func bytesToString(b []byte) string {
	b = bytes.TrimRight(b, "\x00")
	return string(b)
}

func parseNumberFromID(id string) (int, bool) {
	if len(id) < 3 {
		return 0, false
	}

	start := 0
	if len(id) >= 2 && id[0] == '3' && id[1] == '9' {
		start = 2
	}
	if start >= len(id)-1 {
		return 0, false
	}
	core := id[start : len(id)-1]
	var n int
	_, err := fmt.Sscanf(core, "%d", &n)
	return n, err == nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
