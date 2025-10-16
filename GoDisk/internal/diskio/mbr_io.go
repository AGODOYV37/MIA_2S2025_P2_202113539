package diskio

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
)

func ReadMBR(path string) (structs.MBR, error) {
	var m structs.MBR
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return m, err
	}
	if err := binary.Read(f, binary.LittleEndian, &m); err != nil {
		return m, err
	}
	return m, nil
}

func WriteMBR(path string, m structs.MBR) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, &m)
}
