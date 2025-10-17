package ext2

import "errors"

func requireSupportedFS(sb SuperBloque, op string) error {
	if sb.SMagic != MagicEXT2 {
		return errors.New(op + ": superbloque inválido (magic)")
	}
	if sb.SFilesystemType != FileSystemType && sb.SFilesystemType != FileSystemTypeEXT3 {
		return errors.New(op + ": la partición no es EXT2/EXT3 válida (tipo)")
	}
	return nil
}
