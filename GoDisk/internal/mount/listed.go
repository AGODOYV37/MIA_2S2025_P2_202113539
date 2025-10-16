package mount

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type MPView struct {
	ID       string `json:"id"`
	DiskPath string `json:"disk_path"`
	Letter   string `json:"letter"`
	Number   int    `json:"number"`
	PartName string `json:"part_name"`
	Start    int64  `json:"start"`
	Size     int64  `json:"size"`
}

func (r *Registry) MountedPlain() (string, error) {
	views := r.mountedSnapshot()
	if len(views) == 0 {
		return "", ErrNotMounted
	}
	ids := make([]string, len(views))
	for i, v := range views {
		ids[i] = v.ID
	}
	return strings.Join(ids, ", "), nil
}

func (r *Registry) MountedJSON() ([]MPView, error) {
	views := r.mountedSnapshot()
	if len(views) == 0 {
		return nil, ErrNotMounted
	}
	return views, nil
}

func (r *Registry) MountedTable() (string, error) {
	views := r.mountedSnapshot()
	if len(views) == 0 {
		return "", ErrNotMounted
	}

	rows := [][]string{
		{"ID", "DISCO", "LETRA", "NUM", "NOMBRE", "INICIO", "TAM"},
	}
	for _, v := range views {
		rows = append(rows, []string{
			v.ID,
			v.DiskPath,
			v.Letter,
			strconv.Itoa(v.Number),
			v.PartName,
			strconv.FormatInt(v.Start, 10),
			strconv.FormatInt(v.Size, 10),
		})
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for j, col := range row {
			if n := len(col); n > widths[j] {
				widths[j] = n
			}
		}
	}

	var buf bytes.Buffer

	for j, h := range rows[0] {
		buf.WriteString(padRight(h, widths[j]))
		if j < len(rows[0])-1 {
			buf.WriteString("  ")
		}
	}
	buf.WriteByte('\n')

	for j := range rows[0] {
		buf.WriteString(strings.Repeat("-", widths[j]))
		if j < len(rows[0])-1 {
			buf.WriteString("  ")
		}
	}
	buf.WriteByte('\n')

	for i := 1; i < len(rows); i++ {
		for j, col := range rows[i] {
			buf.WriteString(padRight(col, widths[j]))
			if j < len(rows[i])-1 {
				buf.WriteString("  ")
			}
		}
		buf.WriteByte('\n')
	}

	return buf.String(), nil
}

func (r *Registry) mountedSnapshot() []MPView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type item struct {
		letter rune
		number int
		view   MPView
	}
	var items []item

	for _, md := range r.disks {
		for n, mp := range md.byNum {
			if mp == nil {
				continue
			}
			items = append(items, item{
				letter: md.Letter,
				number: n,
				view: MPView{
					ID:       mp.ID,
					DiskPath: mp.DiskPath,
					Letter:   string(md.Letter),
					Number:   mp.Number,
					PartName: mp.PartName,
					Start:    mp.Start,
					Size:     mp.Size,
				},
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].letter == items[j].letter {
			return items[i].number < items[j].number
		}
		return items[i].letter < items[j].letter
	})

	views := make([]MPView, len(items))
	for i, it := range items {
		views[i] = it.view
	}
	return views
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func (v MPView) String() string {
	return fmt.Sprintf("%s(%s:%s%d %s start=%d size=%d)",
		v.ID, v.DiskPath, v.Letter, v.Number, v.PartName, v.Start, v.Size)
}
