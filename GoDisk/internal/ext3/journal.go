package ext3

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/xbin"
)

// Si ya tienes readAt en otro archivo del paquete ext3 y quieres evitar duplicados,
// puedes remover esta función y reutilizar la existente.
func readAt(path string, off int64, data any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	return binary.Read(f, binary.LittleEndian, data)
}

func journalRegion(sb ext2.SuperBloque) (off, entries int64) {
	sbSize := xbin.SizeOf[ext2.SuperBloque]() // tamaño del SB
	jStart := sbSize                          // journal inicia justo después del SB
	jBytes := sb.SBmInodeStart - jStart       // se ubica antes del bitmap de inodos
	entrySz := xbin.SizeOf[structs.Journal]() // tamaño de una entrada de journal
	if entrySz <= 0 || jBytes <= 0 {
		return 0, 0
	}
	return jStart, jBytes / entrySz
}

// Busca el siguiente slot (libre o circular) y asigna JCount secuencial.
func appendJournalEntry(mp *mount.MountedPartition, sb ext2.SuperBloque, entry structs.Journal) error {
	jOff, cap := journalRegion(sb)
	if cap <= 0 {
		// No hay región de journal; no es error.
		return nil
	}

	entrySize := int64(xbin.SizeOf[structs.Journal]())

	// Escanear para slot libre y mayor contador
	var (
		nextIdx   int64 = -1
		lastCount int32 = 0
		cur       structs.Journal
	)

	for i := int64(0); i < cap; i++ {
		off := mp.Start + jOff + i*entrySize
		if err := readAt(mp.DiskPath, off, &cur); err != nil {
			return fmt.Errorf("journal: leyendo entrada %d: %w", i, err)
		}
		if cur.JCount == 0 {
			nextIdx = i // slot libre
			break
		}
		if cur.JCount > lastCount {
			lastCount = cur.JCount
		}
	}
	if nextIdx == -1 {
		// Journal lleno → usar de forma circular
		nextIdx = int64(lastCount % int32(cap))
	}

	// Asignar contador secuencial
	entry.JCount = lastCount + 1

	// Escribir la entrada
	wOff := mp.Start + jOff + nextIdx*entrySize
	if err := writeAt(mp.DiskPath, wOff, entry); err != nil {
		return fmt.Errorf("journal: escribiendo entrada idx=%d: %w", nextIdx, err)
	}
	return nil
}

func AppendJournalIfExt3(reg *mount.Registry, id, op, pth, content string) error {
	mp, ok := reg.GetByID(id)
	if !ok {
		return nil // no montado → no-op
	}

	// Leer SB
	var sb ext2.SuperBloque
	if err := readAt(mp.DiskPath, mp.Start, &sb); err != nil {
		return fmt.Errorf("journal: leyendo SB: %w", err)
	}
	if sb.SFilesystemType != FileSystemTypeExt3 {
		return nil // sólo aplica en EXT3
	}

	// Construir entrada; JCount lo coloca appendJournalEntry
	info := structs.NewInformation(op, pth, content, time.Now())
	entry := structs.Journal{JContent: info}

	return appendJournalEntry(mp, sb, entry)
}

// Versión "wrapper" para compatibilidad — simplemente delega.
func TryAppendJournal(reg *mount.Registry, id, op, pth, content string) error {
	return AppendJournalIfExt3(reg, id, op, pth, content)
}
