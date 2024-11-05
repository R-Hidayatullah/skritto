package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
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
	if file, err := os.Open(filePath); err == nil {
		defer file.Close()

		datFile := &DatFile{}

		// Read DatHeader fields
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

		file.Seek(int64(datFile.Header.MftOffset), io.SeekStart)

		// Read MFTHeader fields
		file.Read(datFile.MFTHeader.Identifier[:])
		datFile.MFTHeader.Unknown, _ = readUint64LE(file)
		datFile.MFTHeader.NumEntries, _ = readUint32LE(file)
		datFile.MFTHeader.UnknownField2, _ = readUint32LE(file)
		datFile.MFTHeader.UnknownField3, _ = readUint32LE(file)

		// Verify MFT magic number
		if string(datFile.MFTHeader.Identifier[:]) != "\x4D\x66\x74\x1A" {
			return nil, fmt.Errorf("invalid MFT header magic number")
		}

		// Read MFTData entries
		datFile.MFTData = make([]MFTData, datFile.MFTHeader.NumEntries)
		for i := range datFile.MFTData {
			datFile.MFTData[i].Offset, _ = readUint64LE(file)
			datFile.MFTData[i].Size, _ = readUint32LE(file)
			datFile.MFTData[i].CompressionFlag, _ = readUint16LE(file)
			datFile.MFTData[i].EntryFlag, _ = readUint16LE(file)
			datFile.MFTData[i].Counter, _ = readUint32LE(file)
			datFile.MFTData[i].CRC, _ = readUint32LE(file)
		}

		// Calculate number of index entries
		numIndexEntries := datFile.MFTData[MftEntryIndexNum].Size / uint32(binary.Size(MFTIndexData{}))
		datFile.MFTIndexData = make([]MFTIndexData, numIndexEntries)

		// Read MFTIndexData entries
		file.Seek(int64(datFile.MFTData[MftEntryIndexNum].Offset), io.SeekStart)
		for i := range datFile.MFTIndexData {
			datFile.MFTIndexData[i].FileID, _ = readUint32LE(file)
			datFile.MFTIndexData[i].BaseID, _ = readUint32LE(file)
		}

		return datFile, nil
	} else {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
}

// Function to extract MFT data by file or base ID
func extractMFTData(datFile *DatFile, number uint32, isFileID bool) ([]byte, error) {
	var index int = -1
	for _, entry := range datFile.MFTIndexData {
		if isFileID && entry.FileID == number {
			index = int(entry.BaseID)
			break
		}
		if !isFileID && entry.BaseID == number {
			index = int(entry.BaseID)
			break
		}
	}
	if index == -1 {
		return nil, fmt.Errorf("MFT entry not found")
	}

	mftEntry := datFile.MFTData[index-1]
	buffer := make([]byte, mftEntry.Size)

	file, err := os.Open("Local.dat")
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to the MFT entry offset in the file
	if _, err := file.Seek(int64(mftEntry.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to MFT entry offset: %w", err)
	}

	// Read data into the buffer
	if _, err := file.Read(buffer); err != nil {
		return nil, fmt.Errorf("failed to read MFT data: %w", err)
	}

	// Decompression (if needed)
	if mftEntry.CompressionFlag != 0 {
		log.Println("File is compressed!")

		var outputBufferSize uint32
		customOutputBufferSize := uint32(0) // Adjust as needed for custom size
		log.Println("Inflate buffer!")

		// Call inflateBuffer to decompress the data
		inflatedData, err := inflateBuffer(uint32(len(buffer)), buffer, &outputBufferSize, customOutputBufferSize)
		if err != nil {
			return nil, fmt.Errorf("decompression failed: %w", err)
		}
		return inflatedData, nil
	}

	return buffer, nil // Return the original buffer if no decompression is needed
}
