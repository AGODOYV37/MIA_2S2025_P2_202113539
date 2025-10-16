package diskio

import "github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/structs"

func FindPrimaryByName(m *structs.MBR, name string) (int, *structs.Partition) {
	for i := 0; i < len(m.Mbr_partitions); i++ {
		p := &m.Mbr_partitions[i]
		if (p.Part_type == 'P' || p.Part_type == 'p') &&
			p.Part_s > 0 &&
			p.Part_start >= 0 &&
			partNameEquals(p, name) {
			return i, p
		}
	}
	return -1, nil
}

func partNameEquals(p *structs.Partition, want string) bool {
	b := p.Part_name[:]

	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	return string(b) == want
}
