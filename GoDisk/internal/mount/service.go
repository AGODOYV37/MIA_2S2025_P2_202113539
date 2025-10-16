package mount

import (
	"fmt"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/diskio"
)

type Service struct {
	reg *Registry
}

func NewService(reg *Registry) *Service {
	return &Service{reg: reg}
}

func (s *Service) Mount(diskPath, partName string) (string, error) {
	diskPath = strings.TrimSpace(diskPath)
	partName = strings.TrimSpace(partName)
	if diskPath == "" || partName == "" {
		return "", Wrap(ErrInvalidArgs, "faltan -path o -name")
	}

	if mp, ok := s.reg.IsMounted(diskPath, partName); ok && mp != nil {
		return "", Wrap(ErrAlreadyMounted, "ya montada: id=%s", mp.ID)
	}

	mbr, err := diskio.ReadMBR(diskPath)
	if err != nil {
		return "", Wrap(ErrMBRRead, "path=%s: %v", diskPath, err)
	}

	idx, p := diskio.FindPrimaryByName(&mbr, partName)
	if idx < 0 || p == nil {
		return "", Wrap(ErrPartitionNotFound, "path=%s name=%s", diskPath, partName)
	}

	letter, err := s.reg.letterForDisk(diskPath)
	if err != nil {
		return "", err
	}
	number, err := s.reg.nextNumForDisk(diskPath)
	if err != nil {
		return "", err
	}
	id := BuildID(number, letter)

	start := p.Part_start
	size := p.Part_s
	mp := &MountedPartition{
		DiskPath: diskPath,
		PartName: partName,
		Letter:   letter,
		Number:   number,
		ID:       id,
		Start:    start,
		Size:     size,
	}
	if err := s.reg.AddMountedPartition(mp); err != nil {
		return "", err
	}

	p.Part_status = '1'
	p.Part_correlative = int32(number)
	copy(p.Part_id[:], []byte(id))

	mbr.Mbr_partitions[idx] = *p

	if err := diskio.WriteMBR(diskPath, mbr); err != nil {
		if removed, _ := s.reg.RemoveByID(id); removed == nil {

		}
		return "", Wrap(ErrMBRWrite, "path=%s: %v", diskPath, err)
	}

	return id, nil
}

func (s *Service) DebugString() string {
	line, err := s.reg.MountedPlain()
	if err != nil {
		return "(sin particiones montadas)"
	}
	return fmt.Sprintf("mounted: %s", line)
}
