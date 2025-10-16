package commands

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
)

func ExecuteMkdisk(size int, unit, fit, path string) error {
	u := strings.ToUpper(strings.TrimSpace(unit))
	var diskSize int64
	switch u {
	case "K":
		diskSize = int64(size) * 1024
	case "M", "":
		diskSize = int64(size) * 1024 * 1024
	default:
		return fmt.Errorf("valor '%s' no válido para -unit (use K o M)", unit)
	}
	if diskSize <= 0 {
		return fmt.Errorf("el parámetro -size debe ser mayor a cero")
	}

	f := strings.ToUpper(strings.TrimSpace(fit))
	var fitByte byte
	switch f {
	case "BF":
		fitByte = 'b'
	case "WF":
		fitByte = 'w'
	case "FF", "":
		fitByte = 'f'
	default:
		return fmt.Errorf("valor '%s' no válido para -fit (use BF, FF o WF)", fit)
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("el parámetro -path es obligatorio")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".mia") {
		path += ".mia"
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return fmt.Errorf("ruta -path inválida")
	}

	rand.Seed(time.Now().UnixNano())
	mbr := structs.NewMBR(diskSize, fitByte, rand.Int63())

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("crear directorios: %w", err)
	}

	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("crear archivo: %w", err)
	}
	cleanup := func(e error) error {
		_ = fh.Close()
		_ = os.Remove(path)
		return e
	}

	if err := fh.Truncate(diskSize); err != nil {
		return cleanup(fmt.Errorf("ajustar tamaño: %w", err))
	}

	if _, err := fh.Seek(0, 0); err != nil {
		return cleanup(fmt.Errorf("seek(0): %w", err))
	}
	if err := binary.Write(fh, binary.LittleEndian, &mbr); err != nil {
		return cleanup(fmt.Errorf("escribiendo MBR: %w", err))
	}

	if err := fh.Close(); err != nil {
		return fmt.Errorf("cerrando archivo: %w", err)
	}
	return nil
}
