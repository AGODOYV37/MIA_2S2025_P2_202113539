package commands

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/structs"
	utils "github.com/AGODOYV37/MIA_2S2025_P2_202113539/pkg"
)

type FdiskOptions struct {
	Path   string // requerido en todos los modos
	Name   string // requerido en delete y add y crear
	Unit   string // b|k|m  (default k). Aplica a size y add
	Type   string // p|e|l  (solo crear, default p)
	Fit    string // bf|ff|wf (solo crear, default wf)
	Size   int64  // >0 al crear
	Delete string // "" | "fast" | "full"
	Add    int64  // puede ser +N o -N
}

func ExecuteFdisk(opt FdiskOptions) error {
	// Normaliza
	opt.Unit = strings.ToLower(strings.TrimSpace(defaultIfEmpty(opt.Unit, "k")))
	opt.Type = strings.ToLower(strings.TrimSpace(defaultIfEmpty(opt.Type, "p")))
	opt.Fit = strings.ToLower(strings.TrimSpace(defaultIfEmpty(opt.Fit, "wf")))
	opt.Delete = strings.ToLower(strings.TrimSpace(opt.Delete))

	// Validaciones de modo
	switch {
	case opt.Delete != "":
		if opt.Name == "" {
			return errors.New("fdisk delete: -name requerido")
		}
		if opt.Delete != "fast" && opt.Delete != "full" {
			return errors.New("fdisk delete: -delete debe ser fast|full") // Full rellena con \0.
		}
	case opt.Add != 0:
		if opt.Name == "" {
			return errors.New("fdisk add: -name requerido")
		}
	default: // crear
		if opt.Name == "" || opt.Size <= 0 {
			return errors.New("fdisk create: se requieren -name y -size>0")
		}
	}

	// Abre disco
	file, err := os.OpenFile(opt.Path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("fdisk: disco no existe: %s", opt.Path)
		}
		return fmt.Errorf("fdisk: error al abrir disco: %w", err)
	}
	defer file.Close()

	// Lee MBR
	mbr, err := utils.ReadMBR(file)
	if err != nil {
		return fmt.Errorf("fdisk: error leyendo MBR: %w", err)
	}

	// Ejecutar por modo
	switch {
	case opt.Delete != "":
		return deletePartition(file, &mbr, opt)
	case opt.Add != 0:
		return addSpace(file, &mbr, opt)
	default:
		return createPartition(file, &mbr, opt)
	}
}

func createPartition(file *os.File, mbr *structs.MBR, opt FdiskOptions) error {
	// Convierte size
	size := toBytes(opt.Size, opt.Unit)

	switch opt.Type {
	case "p":
		createPrimary(file, mbr, opt.Name, opt.Fit, size)
	case "e":
		createExtended(file, mbr, opt.Name, opt.Fit, size)
	case "l":
		createLogical(file, mbr, opt.Name, opt.Fit, size)
	default:
		return fmt.Errorf("fdisk create: tipo inválido %q (p/e/l)", opt.Type)
	}

	// Guardar MBR si createXxx lo modificó
	if err := utils.WriteMBR(file, mbr); err != nil {
		return err
	}
	return nil
}

func deletePartition(file *os.File, mbr *structs.MBR, opt FdiskOptions) error {
	// 1) Primaria/Extendida por nombre en MBR
	for i := 0; i < 4; i++ {
		p := &mbr.Mbr_partitions[i]
		if p.Part_status == '1' && strings.Trim(string(p.Part_name[:]), "\x00") == opt.Name {
			// Marcar libre
			start := p.Part_start
			size := p.Part_s
			p.Part_status = '0'
			p.Part_type = 0
			p.Part_fit = 0
			p.Part_start = 0
			p.Part_s = 0
			clearName(p.Part_name[:])

			if err := utils.WriteMBR(file, mbr); err != nil {
				return err
			}
			// Full: rellena con \0 el área liberada.
			if opt.Delete == "full" {
				if err := zeroRegion(file, start, size); err != nil {
					return err
				}
			}
			fmt.Printf("Partición '%s' eliminada (%s).\n", opt.Name, opt.Delete)
			return nil
		}
	}

	// 2) Buscar en LÓGICAS (EBRs) dentro de la extendida
	// 2) Buscar en LÓGICAS (EBRs) dentro de la extendida
	ext, ok := findExtended(*mbr)
	if !ok {
		return fmt.Errorf("fdisk delete: no existe partición '%s'", opt.Name)
	}

	ebrSize := int64(binary.Size(structs.EBR{}))
	headAddr := ext.Part_start

	// Recorre cadena EBR manteniendo dirección del anterior
	var prevAddr int64 = -1
	curAddr := headAddr

	for {
		cur, err := utils.ReadEBR(file, curAddr)
		if err != nil {
			return fmt.Errorf("fdisk delete: leer EBR: %w", err)
		}
		// Caso head vacío sin lógicas
		if curAddr == headAddr && cur.Part_status != '1' && cur.Part_next == -1 {
			return fmt.Errorf("fdisk delete: no existen lógicas")
		}

		curName := strings.Trim(string(cur.Part_name[:]), "\x00")
		if cur.Part_status == '1' && curName == opt.Name {
			// === Encontrada la lógica a eliminar ===
			if prevAddr == -1 {
				// Eliminar la PRIMERA lógica (en headAddr)
				if cur.Part_next == -1 {
					// Era la única: restaurar head vacío
					empty := structs.EBR{Part_status: '0', Part_next: -1}
					if err := utils.WriteEBR(file, &empty, headAddr); err != nil {
						return err
					}
				} else {
					// Hay más lógicas: traer el siguiente EBR y sobrescribir el head
					next, err := utils.ReadEBR(file, cur.Part_next)
					if err != nil {
						return err
					}
					if err := utils.WriteEBR(file, &next, headAddr); err != nil {
						return err
					}
				}
				// Opcionalmente limpia área de datos/EBR del eliminado:
				if opt.Delete == "full" {
					// limpia EBR original (headAddr) ya fue sobrescrito; limpia datos de la partición
					if err := zeroRegion(file, cur.Part_start, cur.Part_s); err != nil {
						return err
					}
				} else {
					// Marca como vacío por seguridad (aunque ya sobreescribimos/ajustamos)
					cur.Part_status = '0'
					if err := utils.WriteEBR(file, &cur, headAddr); err != nil {
						return err
					}
				}
			} else {
				// Eliminar una lógica intermedia/final: encadenar prev -> cur.next
				prev, err := utils.ReadEBR(file, prevAddr)
				if err != nil {
					return err
				}
				prev.Part_next = cur.Part_next
				if err := utils.WriteEBR(file, &prev, prevAddr); err != nil {
					return err
				}

				if opt.Delete == "full" {
					// Limpia EBR + datos de la lógica eliminada
					if err := zeroRegion(file, curAddr, ebrSize); err != nil {
						return err
					}
					if err := zeroRegion(file, cur.Part_start, cur.Part_s); err != nil {
						return err
					}
				} else {
					// Marca como vacío por seguridad
					cur.Part_status = '0'
					if err := utils.WriteEBR(file, &cur, curAddr); err != nil {
						return err
					}
				}
			}

			fmt.Printf("Partición lógica '%s' eliminada (%s).\n", opt.Name, opt.Delete)
			return nil
		}

		if cur.Part_next == -1 {
			break
		}
		prevAddr = curAddr
		curAddr = cur.Part_next
	}

	return fmt.Errorf("fdisk delete: no existe partición '%s'", opt.Name)

}

func addSpace(file *os.File, mbr *structs.MBR, opt FdiskOptions) error {
	delta := toBytes(opt.Add, opt.Unit) // puede ser negativo
	if delta == 0 {
		return nil
	}

	// 1) Intentar en primarias/extendida
	for i := 0; i < 4; i++ {
		p := &mbr.Mbr_partitions[i]
		if p.Part_status == '1' && strings.Trim(string(p.Part_name[:]), "\x00") == opt.Name {
			if delta > 0 {
				// Validar espacio contiguo después.
				free := utils.GetFreeSpaces(mbr)
				end := p.Part_start + p.Part_s
				ok, maxGrow := contiguousAfter(end, free)
				if !ok || maxGrow < delta {
					return fmt.Errorf("fdisk add: no hay espacio libre contiguo suficiente después de '%s'", opt.Name)
				}
				p.Part_s += delta
			} else {
				// Reducir: que no quede tamaño negativo.
				newSize := p.Part_s + delta
				if newSize <= 0 {
					return fmt.Errorf("fdisk add: tamaño resultante inválido")
				}
				p.Part_s = newSize
			}
			if err := utils.WriteMBR(file, mbr); err != nil {
				return err
			}
			fmt.Printf("Partición '%s' redimensionada (%+d %s).\n", opt.Name, opt.Add, strings.ToUpper(opt.Unit))
			return nil
		}
	}

	// 2) Intentar en lógicas
	ext, hasExt := findExtended(*mbr)
	if !hasExt {
		return fmt.Errorf("fdisk add: no existe partición '%s'", opt.Name)
	}

	cur, err := utils.ReadEBR(file, ext.Part_start)
	if err != nil {
		return fmt.Errorf("fdisk add: leer EBR: %w", err)
	}
	if cur.Part_status != '1' && cur.Part_next == -1 {
		return fmt.Errorf("fdisk add: no existen lógicas")
	}
	for {
		curName := strings.Trim(string(cur.Part_name[:]), "\x00")
		if cur.Part_status == '1' && curName == opt.Name {
			ebrSize := int64(binary.Size(cur))
			if delta > 0 {
				// hay que validar hueco hasta el siguiente EBR o hasta fin de extendida
				nextStart := ext.Part_start + ext.Part_s
				if cur.Part_next != -1 {
					nextStart = cur.Part_next
				}
				curEnd := cur.Part_start + cur.Part_s
				disp := nextStart - curEnd
				if disp < delta {
					return fmt.Errorf("fdisk add: no hay espacio contiguo suficiente en extendida")
				}
				cur.Part_s += delta
			} else {
				newSize := cur.Part_s + delta
				if newSize <= ebrSize {
					return fmt.Errorf("fdisk add: tamaño resultante inválido")
				}
				cur.Part_s = newSize
			}
			// reescribir EBR en su dirección física (EBR inicia en startEBR = Part_start - sizeof(EBR))
			startEBR := cur.Part_start - ebrSize
			if err := utils.WriteEBR(file, &cur, startEBR); err != nil {
				return err
			}
			fmt.Printf("Partición lógica '%s' redimensionada (%+d %s).\n", opt.Name, opt.Add, strings.ToUpper(opt.Unit))
			return nil
		}
		if cur.Part_next == -1 {
			break
		}
		cur, err = utils.ReadEBR(file, cur.Part_next)
		if err != nil {
			return fmt.Errorf("fdisk add: leer EBR: %w", err)
		}
	}
	return fmt.Errorf("fdisk add: no existe partición '%s'", opt.Name)
}

func toBytes(n int64, unit string) int64 {
	switch strings.ToLower(unit) {
	case "b":
		return n
	case "m":
		return n * 1024 * 1024
	default:
		return n * 1024 // k por defecto (enunciado).
	}
}

func contiguousAfter(end int64, spaces []utils.FreeSpace) (bool, int64) {
	for _, s := range spaces {
		if s.Start == end {
			return true, s.Size
		}
	}
	return false, 0
}

func findExtended(mbr structs.MBR) (structs.Partition, bool) {
	for _, p := range mbr.Mbr_partitions {
		if p.Part_status == '1' && p.Part_type == 'E' {
			return p, true
		}
	}
	return structs.Partition{}, false
}

func zeroRegion(file *os.File, start, size int64) error {
	if size <= 0 {
		return nil
	}
	if _, err := file.Seek(start, 0); err != nil {
		return err
	}
	chunk := make([]byte, 1024*64)
	toWrite := size
	for toWrite > 0 {
		n := int64(len(chunk))
		if n > toWrite {
			n = toWrite
		}
		if _, err := file.Write(chunk[:n]); err != nil {
			return err
		}
		toWrite -= n
	}
	return nil
}

func clearName(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func defaultIfEmpty(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func createPrimary(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Primaria...")

	//  Validaciones
	partitionCount := 0
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			partitionCount++

			if strings.Trim(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00") == name {
				fmt.Printf("Error: ya existe una partición con el nombre '%s'.\n", name)
				return
			}
		}
	}
	if partitionCount >= 4 {
		fmt.Println("Error: ya existen 4 particiones, no se pueden crear más.")
		return
	}

	freeSpaces := utils.GetFreeSpaces(mbr)
	var bestFitStart int64 = -1

	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio contiguo para la partición.")
		return
	}

	var newPartition structs.Partition
	newPartition.Part_status = '1'
	newPartition.Part_type = 'P'
	newPartition.Part_fit = byte(strings.ToUpper(fit)[0])
	newPartition.Part_start = bestFitStart
	newPartition.Part_s = size
	copy(newPartition.Part_name[:], name)

	added := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '0' {
			mbr.Mbr_partitions[i] = newPartition
			added = true
			break
		}
	}
	if !added {
		fmt.Println("Error: no se encontró un slot de partición libre.")
		return
	}

	err := utils.WriteMBR(file, mbr)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Partición primaria '%s' creada exitosamente.\n", name)
}

func createExtended(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Extendida...")

	partitionCount := 0
	hasExtended := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			partitionCount++
			if mbr.Mbr_partitions[i].Part_type == 'E' {
				hasExtended = true
			}
			if strings.Trim(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00") == name {
				fmt.Printf("Error: ya existe una partición con el nombre '%s'.\n", name)
				return
			}
		}
	}
	if partitionCount >= 4 {
		fmt.Println("Error: ya existen 4 particiones, no se pueden crear más.")
		return
	}
	if hasExtended {
		fmt.Println("Error: ya existe una partición extendida en este disco.")
		return
	}

	freeSpaces := utils.GetFreeSpaces(mbr)
	var bestFitStart int64 = -1

	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio contiguo para la partición.")
		return
	}

	var newPartition structs.Partition
	newPartition.Part_status = '1'
	newPartition.Part_type = 'E'
	newPartition.Part_fit = byte(strings.ToUpper(fit)[0])
	newPartition.Part_start = bestFitStart
	newPartition.Part_s = size
	copy(newPartition.Part_name[:], name)

	added := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '0' {
			mbr.Mbr_partitions[i] = newPartition
			added = true
			break
		}
	}
	if !added {
		fmt.Println("Error: no se encontró un slot de partición libre.")
		return
	}

	err := utils.WriteMBR(file, mbr)
	if err != nil {
		fmt.Println(err)
		return
	}

	var firstEBR structs.EBR
	firstEBR.Part_status = '0'
	firstEBR.Part_next = -1
	err = utils.WriteEBR(file, &firstEBR, newPartition.Part_start)
	if err != nil {
		fmt.Printf("Error al inicializar el primer EBR: %v\n", err)
		return
	}

	fmt.Printf("Partición extendida '%s' creada exitosamente.\n", name)
}

func createLogical(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Lógica...")

	var extendedPartition structs.Partition
	foundExtended := false
	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type == 'E' {
			extendedPartition = mbr.Mbr_partitions[i]
			foundExtended = true
			break
		}
	}

	if !foundExtended {
		fmt.Println("Error: No se puede crear una partición lógica porque no existe una partición extendida.")
		return
	}

	var logicalPartitions []structs.EBR
	currentEBR, err := utils.ReadEBR(file, extendedPartition.Part_start)
	if err != nil {
		fmt.Printf("Error al leer el primer EBR: %v\n", err)
		return
	}
	lastEBRAddress := extendedPartition.Part_start

	if currentEBR.Part_status == '1' {
		logicalPartitions = append(logicalPartitions, currentEBR)
		for currentEBR.Part_next != -1 {
			lastEBRAddress = currentEBR.Part_next
			currentEBR, err = utils.ReadEBR(file, currentEBR.Part_next)
			if err != nil {
				fmt.Printf("Error al leer la cadena de EBRs: %v\n", err)
				return
			}
			logicalPartitions = append(logicalPartitions, currentEBR)
		}
	}

	freeSpaces := utils.GetFreeSpacesInExtended(extendedPartition, logicalPartitions)
	var bestFitStart int64 = -1

	ebrSize := int64(binary.Size(structs.EBR{}))
	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size+ebrSize)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size+ebrSize)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size+ebrSize)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio en la partición extendida.")
		return
	}

	var newEBR structs.EBR
	newEBR.Part_status = '1'
	newEBR.Part_fit = byte(strings.ToUpper(fit)[0])
	newEBR.Part_start = bestFitStart + ebrSize
	newEBR.Part_s = size
	newEBR.Part_next = -1
	copy(newEBR.Part_name[:], name)

	err = utils.WriteEBR(file, &newEBR, bestFitStart)
	if err != nil {
		fmt.Printf("Error al escribir el nuevo EBR: %v\n", err)
		return
	}

	if currentEBR.Part_status == '1' {
		currentEBR.Part_next = bestFitStart
		err = utils.WriteEBR(file, &currentEBR, lastEBRAddress)
		if err != nil {
			fmt.Printf("Error al actualizar el último EBR: %v\n", err)
			return
		}
	} else {
		err = utils.WriteEBR(file, &newEBR, extendedPartition.Part_start)
		if err != nil {
			fmt.Printf("Error al escribir el primer EBR lógico: %v\n", err)
			return
		}
	}

	fmt.Printf("Partición lógica '%s' creada exitosamente.\n", name)
}
