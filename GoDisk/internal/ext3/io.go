package ext3

import (
	"encoding/binary"
	"io"
	"os"
)

func writeAt(path string, off int64, data any) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, data)
}

func writeBytes(path string, off int64, buf []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	_, err = f.Write(buf)
	return err
}
