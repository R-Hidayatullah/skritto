package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/k0kubun/pp/v3"
)

const (
	DatMagicNumber   = 3
	MftMagicNumber   = 4
	MftEntryIndexNum = 1
)

type DatHeader struct {
	Version       uint8
	Identifier    [DatMagicNumber]uint8
	HeaderSize    uint32
	UnknownField  uint32
	ChunkSize     uint32
	CRC           uint32
	UnknownField2 uint32
	MftOffset     uint64
	MftSize       uint32
	Flags         uint32
}

type MFTHeader struct {
	Identifier    [MftMagicNumber]uint8
	Unknown       uint64
	NumEntries    uint32
	UnknownField2 uint32
	UnknownField3 uint32
}

type MFTData struct {
	Offset          uint64
	Size            uint32
	CompressionFlag uint16
	EntryFlag       uint16
	Counter         uint32
	CRC             uint32
}

type MFTIndexData struct {
	FileID uint32
	BaseID uint32
}

type DatFile struct {
	Header       DatHeader
	MFTHeader    MFTHeader
	MFTData      []MFTData
	MFTIndexData []MFTIndexData
}

// Helper function to read little-endian values
func readUint16LE(r io.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(r, binary.LittleEndian, &value)
	return value, err
}

func readUint32LE(r io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(r, binary.LittleEndian, &value)
	return value, err
}

func readUint64LE(r io.Reader) (uint64, error) {
	var value uint64
	err := binary.Read(r, binary.LittleEndian, &value)
	return value, err
}

// Function to load .dat file and populate DatFile structure
func loadDatFile(filePath string) (*DatFile, error) {
	log.Printf("Opening .dat file: %s\n", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open .dat file: %v\n", err)
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	log.Println("Reading DatHeader...")
	datFile := &DatFile{}
	binary.Read(file, binary.LittleEndian, &datFile.Header.Version)
	file.Read(datFile.Header.Identifier[:])
	datFile.Header.HeaderSize, _ = readUint32LE(file)
	datFile.Header.UnknownField, _ = readUint32LE(file)
	datFile.Header.ChunkSize, _ = readUint32LE(file)
	datFile.Header.CRC, _ = readUint32LE(file)
	datFile.Header.UnknownField2, _ = readUint32LE(file)
	datFile.Header.MftOffset, _ = readUint64LE(file)
	datFile.Header.MftSize, _ = readUint32LE(file)
	datFile.Header.Flags, _ = readUint32LE(file)

	log.Printf("Seeking to MFT offset: %d\n", datFile.Header.MftOffset)
	file.Seek(int64(datFile.Header.MftOffset), io.SeekStart)

	log.Println("Reading MFTHeader...")
	file.Read(datFile.MFTHeader.Identifier[:])
	datFile.MFTHeader.Unknown, _ = readUint64LE(file)
	datFile.MFTHeader.NumEntries, _ = readUint32LE(file)
	datFile.MFTHeader.UnknownField2, _ = readUint32LE(file)
	datFile.MFTHeader.UnknownField3, _ = readUint32LE(file)

	log.Println("Verifying MFT magic number...")
	if string(datFile.MFTHeader.Identifier[:]) != "\x4D\x66\x74\x1A" {
		log.Println("Invalid MFT header magic number.")
		return nil, fmt.Errorf("invalid MFT header magic number")
	}

	log.Printf("Reading %d MFTData entries...\n", datFile.MFTHeader.NumEntries)
	datFile.MFTData = make([]MFTData, datFile.MFTHeader.NumEntries)
	for i := range datFile.MFTData {
		datFile.MFTData[i].Offset, _ = readUint64LE(file)
		datFile.MFTData[i].Size, _ = readUint32LE(file)
		datFile.MFTData[i].CompressionFlag, _ = readUint16LE(file)
		datFile.MFTData[i].EntryFlag, _ = readUint16LE(file)
		datFile.MFTData[i].Counter, _ = readUint32LE(file)
		datFile.MFTData[i].CRC, _ = readUint32LE(file)
	}

	log.Println("Calculating number of MFT index entries...")
	numIndexEntries := datFile.MFTData[MftEntryIndexNum].Size / uint32(binary.Size(MFTIndexData{}))
	datFile.MFTIndexData = make([]MFTIndexData, numIndexEntries)

	log.Println("Parsing MFT index data...")
	file.Seek(int64(datFile.MFTData[MftEntryIndexNum].Offset), io.SeekStart)
	for i := range datFile.MFTIndexData {
		datFile.MFTIndexData[i].FileID, _ = readUint32LE(file)
		datFile.MFTIndexData[i].BaseID, _ = readUint32LE(file)
	}

	return datFile, nil
}

// Function to extract MFT data by file or base ID
func extractMFTData(datFile *DatFile, number uint32, isFileID bool) ([]byte, error) {
	log.Printf("Starting MFT data extraction for number: %d, isFileID: %v\n", number, isFileID)
	var index int = -1
	for _, entry := range datFile.MFTIndexData {
		if isFileID && entry.FileID == number {
			index = int(entry.BaseID)
			pp.Println(entry)
			break
		}
		if !isFileID && entry.BaseID == number {
			index = int(entry.BaseID)
			pp.Println(entry)
			break
		}
	}
	if index == -1 {
		log.Println("MFT entry not found.")
		return nil, fmt.Errorf("MFT entry not found")
	}

	log.Printf("Located MFT entry at index %d.\n", index)
	mftEntry := datFile.MFTData[index-1]
	pp.Println(mftEntry)
	buffer := make([]byte, mftEntry.Size)

	log.Printf("Opening .dat file to read MFT entry data...\n")
	file, err := os.Open("C:\\Program Files (x86)\\Steam\\steamapps\\common\\Guild Wars 2\\Gw2.dat")
	if err != nil {
		log.Printf("Failed to open .dat file: %v\n", err)
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	log.Printf("Seeking to MFT entry offset: %d\n", mftEntry.Offset)
	if _, err := file.Seek(int64(mftEntry.Offset), io.SeekStart); err != nil {
		log.Printf("Failed to seek to MFT entry offset: %v\n", err)
		return nil, fmt.Errorf("failed to seek to MFT entry offset: %w", err)
	}

	log.Printf("Reading %d bytes of MFT entry data...\n", mftEntry.Size)
	if _, err := file.Read(buffer); err != nil {
		log.Printf("Failed to read MFT data: %v\n", err)
		return nil, fmt.Errorf("failed to read MFT data: %w", err)
	}

	if mftEntry.CompressionFlag != 0 {
		log.Println("Detected compressed MFT entry data.")

		var outputBufferSize uint32
		customOutputBufferSize := uint32(0) // Adjust as needed for custom size
		log.Println("Attempting to decompress MFT entry data...")

		inflatedData, err := inflateBuffer(buffer, &outputBufferSize, customOutputBufferSize)
		if err != nil {
			log.Printf("Decompression failed: %v\n", err)
			return nil, fmt.Errorf("decompression failed: %w", err)
		}
		log.Println("Decompression successful.")
		return inflatedData, nil
	}

	log.Println("Returning uncompressed MFT entry data.")
	return buffer, nil
}
