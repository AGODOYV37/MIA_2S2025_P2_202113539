package structs

import (
	"time"
)

// Information representa el contenido de una entrada del journal.
// i_operation: operación realizada (máx 10 chars)
// i_path: ruta donde se realizó (máx 32 chars)
// i_content: contenido (si aplica, máx 64 chars)
// i_date: fecha/hora en float64 (por ej. epoch seconds con fracción)
type Information struct {
	I_operation [10]byte
	I_path      [32]byte
	I_content   [64]byte
	I_date      float64
}

// Journal es una entrada de la bitácora.
// j_count: contador (ordinal) de la entrada
// j_content: información asociada
type Journal struct {
	JCount   int32
	JContent Information
}

// ----------------- Helpers opcionales -----------------

// NewInformation construye Information truncando/padding los campos de texto.
func NewInformation(operation, path, content string, when time.Time) Information {
	var info Information
	setFixed(info.I_operation[:], operation)
	setFixed(info.I_path[:], path)
	setFixed(info.I_content[:], content)
	info.I_date = float64(when.UnixNano()) / 1e9 // segundos con fracción
	return info
}

// NewJournal crea una entrada Journal con contador y contenido dados.
func NewJournal(count int32, info Information) Journal {
	return Journal{
		JCount:   count,
		JContent: info,
	}
}

// setFixed copia s a un buffer fijo b, truncando si excede y dejando padding en cero.
func setFixed(b []byte, s string) {
	n := copy(b, []byte(s))
	// Relleno en cero si sobran bytes (el array ya nace en cero, esto es defensivo)
	for i := n; i < len(b); i++ {
		b[i] = 0
	}
}
