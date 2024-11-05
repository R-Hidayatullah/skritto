package main

import (
	"encoding/hex"
	"fmt"

	"github.com/k0kubun/pp/v3"
)

func main() {
	datFile, err := loadDatFile("Local.dat")
	if err != nil {
		fmt.Println("Error loading .dat file:", err)
		return
	}
	pp.Println(&datFile.Header)
	data, err := extractMFTData(datFile, 16, false)
	if err != nil {
		fmt.Println("Error extracting MFT data:", err)
		return
	}

	fmt.Printf("Extracted data (first 128 bytes) :\n%s", hex.Dump(data[:128]))
}
