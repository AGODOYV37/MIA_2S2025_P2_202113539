package ext3

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
)

type JournalRow struct {
	Count     int32  `json:"count"`
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	Date      string `json:"date"` // RFC3339
}

// ListJournal devuelve las entradas del journal como arreglo de filas listas para serializar.
func ListJournal(reg *mount.Registry, id string) ([]JournalRow, error) {
	mp, ok := reg.GetByID(id)
	if !ok {
		return nil, fmt.Errorf("journaling: id %s no está montado", id)
	}

	var sb ext2.SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return nil, fmt.Errorf("journaling: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemTypeExt3 {
		return nil, fmt.Errorf("journaling: solo aplica para particiones EXT3")
	}

	entries, err := readAllJournalEntries(mp, sb) // función ya existe en ext3/recovery.go
	if err != nil {
		return nil, err
	}

	out := make([]JournalRow, 0, len(entries))
	for _, e := range entries {
		op := strings.ToUpper(strings.TrimSpace(trimNull(e.JContent.I_operation[:])))
		p := strings.TrimSpace(trimNull(e.JContent.I_path[:]))
		c := strings.TrimSpace(trimNull(e.JContent.I_content[:]))
		ts := int64(e.JContent.I_date)
		if ts < 0 {
			ts = 0
		}
		out = append(out, JournalRow{
			Count:     e.JCount,
			Operation: op,
			Path:      p,
			Content:   c,
			Date:      time.Unix(ts, 0).UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

// ListJournalJSON devuelve el JSON formateado de las entradas del journal.
func ListJournalJSON(reg *mount.Registry, id string) (string, error) {
	rows, err := ListJournal(reg, id)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("journaling: serializando JSON: %w", err)
	}
	return string(b), nil
}
