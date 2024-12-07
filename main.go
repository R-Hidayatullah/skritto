package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/k0kubun/pp/v3"
)

func main() {
	// Retrieve command-line arguments
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: program <MFT index>")
		return
	}

	// Convert the MFT index argument to uint32
	mftIndex, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		fmt.Printf("Error parsing MFT index '%s': %v\n", args[1], err)
		return
	}

	// Load the .dat file
	log.Println("Attempting to load .dat file...")
	datFilePath := "C:\\Program Files (x86)\\Steam\\steamapps\\common\\Guild Wars 2\\Gw2.dat"
	log.Printf("Loading .dat file from path: %s\n", datFilePath)
	datFile, err := loadDatFile(datFilePath)
	if err != nil {
		fmt.Printf("Error loading .dat file: %v\n", err)
		return
	}
	log.Println(".dat file loaded successfully.")
	pp.Println(&datFile.Header)

	// Extract MFT data
	log.Printf("Attempting to extract MFT data for index %d...\n", mftIndex)
	data, err := extractMFTData(datFile, uint32(mftIndex), false)
	if err != nil {
		fmt.Printf("Error extracting MFT data for index %d: %v\n", mftIndex, err)
		return
	}

	log.Printf("Successfully extracted MFT data for index %d.\n", mftIndex)
	fmt.Printf("Extracted data (first 128 bytes):\n%s\n", hex.Dump(data[:128]))
}
