package reports

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/mount"
)

type Name string

const (
	ReportMBR     Name = "mbr"
	ReportDisk    Name = "disk"
	ReportInode   Name = "inode"
	ReportInodes  Name = "inodes"
	ReportBlock   Name = "block"
	ReportBlocks  Name = "blocks"
	ReportBmInode Name = "bm_inode"
	ReportBmBlock Name = "bm_block"
	ReportTree    Name = "tree"
	ReportSB      Name = "sb"
	ReportFile    Name = "file"
	ReportLS      Name = "ls"
)

type Params struct {
	ID   string
	Name Name
	Path string
	Ruta string
}

func (p *Params) Clean() {
	p.ID = strings.TrimSpace(p.ID)
	p.Path = filepath.Clean(strings.TrimSpace(p.Path))
	p.Ruta = strings.TrimSpace(p.Ruta)
	p.Name = Name(strings.ToLower(strings.TrimSpace(string(p.Name))))
}

func (p Params) Validate() error {
	if p.ID == "" {
		return errors.New("rep: -id requerido")
	}
	if p.Name == "" {
		return errors.New("rep: -name requerido")
	}
	if p.Path == "" {
		return errors.New("rep: -path requerido")
	}
	return nil
}

func Generate(reg *mount.Registry, p Params) error {
	switch p.Name {
	case ReportMBR:
		return GenerateMBR(reg, p.ID, p.Path)
	case ReportDisk:
		return GenerateDisk(reg, p.ID, p.Path)
	case ReportInode:
		return GenerateInode(reg, p.ID, p.Ruta, p.Path)
	case ReportInodes:
		return GenerateInodes(reg, p.ID, p.Path)
	case ReportBlock:
		return GenerateBlock(reg, p.ID, p.Path)
	case ReportBmInode:
		return GenerateBmInode(reg, p.ID, p.Path)
	case ReportBmBlock:
		return GenerateBmBlock(reg, p.ID, p.Path)
	case ReportTree:
		return GenerateTree(reg, p.ID, p.Path)
	case ReportFile:
		return GenerateFile(reg, p.ID, p.Ruta, p.Path)
	case ReportLS:
		return GenerateLS(reg, p.ID, p.Ruta, p.Path)
	default:
		return errors.New("rep: reporte no soportado: " + string(p.Name))
	}
}
