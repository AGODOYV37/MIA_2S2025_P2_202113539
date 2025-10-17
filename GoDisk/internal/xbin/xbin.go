package xbin

import "encoding/binary"

func SizeOf[T any]() int64 {
	var z T
	return int64(binary.Size(z))
}
