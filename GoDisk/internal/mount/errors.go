package mount

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidArgs = errors.New("mount: argumentos inválidos")

	ErrPartitionNotFound = errors.New("mount: la partición no existe")
	ErrNotPrimary        = errors.New("mount: la partición no es primaria")

	ErrAlreadyMounted = errors.New("mount: la partición ya está montada")
	ErrIDNotFound     = errors.New("mount: ID no encontrado")
	ErrNotMounted     = errors.New("mount: no hay particiones montadas")

	ErrDiskLetterExhausted = errors.New("mount: no hay letras disponibles para nuevos discos")

	ErrMBRRead  = errors.New("mount: no se pudo leer el MBR")
	ErrMBRWrite = errors.New("mount: no se pudo escribir el MBR")
)

func Wrap(base error, format string, a ...any) error {
	if base == nil {
		return nil
	}
	if format == "" {
		return base
	}
	return fmt.Errorf("%w: %s", base, fmt.Sprintf(format, a...))
}

func IsInvalidArgs(err error) bool       { return errors.Is(err, ErrInvalidArgs) }
func IsPartitionNotFound(err error) bool { return errors.Is(err, ErrPartitionNotFound) }
func IsNotPrimary(err error) bool        { return errors.Is(err, ErrNotPrimary) }
func IsAlreadyMounted(err error) bool    { return errors.Is(err, ErrAlreadyMounted) }
func IsIDNotFound(err error) bool        { return errors.Is(err, ErrIDNotFound) }
func IsNotMounted(err error) bool        { return errors.Is(err, ErrNotMounted) }
func IsLetterExhausted(err error) bool   { return errors.Is(err, ErrDiskLetterExhausted) }
func IsMBRRead(err error) bool           { return errors.Is(err, ErrMBRRead) }
func IsMBRWrite(err error) bool          { return errors.Is(err, ErrMBRWrite) }
