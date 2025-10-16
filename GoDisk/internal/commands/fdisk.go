package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/AGODOYV37/MIA_2S2025_P1_202113539/internal/structs"
	utils "github.com/AGODOYV37/MIA_2S2025_P1_202113539/pkg"
)

func ExecuteFdisk(path, name, unit, typeStr, fit string, size int64) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Error: el disco en la ruta '%s' no existe.\n", path)
		} else {
			fmt.Printf("Error al abrir el disco: %v\n", err)
		}
		return
	}
	defer file.Close()

	var mbr structs.MBR
	file.Seek(0, 0)
	err = binary.Read(file, binary.LittleEndian, &mbr)
	if err != nil {
		fmt.Printf("Error al leer el MBR del disco: %v\n", err)
		return
	}

	var partitionSize int64
	switch strings.ToLower(unit) {
	case "b":
		partitionSize = size
	case "m":
		partitionSize = size * 1024 * 1024
	default:
		partitionSize = size * 1024
	}

	switch strings.ToLower(typeStr) {
	case "p":
		createPrimary(file, &mbr, name, fit, partitionSize)
	case "e":
		createExtended(file, &mbr, name, fit, partitionSize)
	case "l":
		createLogical(file, &mbr, name, fit, partitionSize)
	default:
		fmt.Printf("Error: tipo de partición '%s' no reconocido.\n", typeStr)
	}
}

type freeSpace struct {
	start int64
	end   int64
	size  int64
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
