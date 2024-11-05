package main

import (
	"errors"
	"fmt"
	"log"
	"os"
)

const (
	MAX_SYMBOL_VALUE     = 285
	MAX_CODE_BITS_LENGTH = 32
	BlockSize            = 0x4000 // Define block size constant
)

// HuffmanTree structure
type HuffmanTree struct {
	SymbolValues      [MAX_SYMBOL_VALUE]uint16
	CompressedCodes   [MAX_SYMBOL_VALUE]uint32
	BitsLength        [MAX_SYMBOL_VALUE]uint8
	SymbolValueOffset [MAX_SYMBOL_VALUE]uint16
}

// State structure for managing decompression
type State struct {
	InputData     []uint32 // Input data (compressed)
	InputSize     uint32   // Size of the input data
	InputPosition uint32   // Current position in the input
	Head          uint32   // Head for reading bits
	Bits          uint32   // Bits read from input
	Buffer        uint32   // Buffer for storing bits
	Empty         bool     // Flag to check if input is empty
}

var (
	huffmanTreeDictInitialized bool
	huffmanTreeDict            HuffmanTree // Assume this is a defined structure for your Huffman tree
)

// pullByte pulls a byte from the input data
func pullByte(stateData *State) {
	if stateData.Bits >= 32 {
		log.Fatal("Tried to pull a value while we still have 32 bits available.")
		return
	}

	if (stateData.InputPosition+1)%BlockSize == 0 {
		stateData.InputPosition++
	}

	if stateData.InputPosition >= stateData.InputSize {
		log.Fatal("Reached end of input while trying to fetch a new byte.")
		return
	}

	tempValue := stateData.InputData[stateData.InputPosition]

	if stateData.Bits == 0 {
		stateData.Head = tempValue
		stateData.Buffer = 0
	} else {
		stateData.Head |= tempValue >> stateData.Bits
		stateData.Buffer = tempValue << (32 - stateData.Bits)
	}

	stateData.Bits += 32
	stateData.InputPosition++
}

// needBits ensures we have enough bits
func needBits(stateData *State, bits uint8) {
	if bits > 32 {
		log.Fatal("Tried to need more than 32 bits.")
	}

	if stateData.Bits < uint32(bits) {
		pullByte(stateData)
	}
}

// dropBits drops a specified number of bits
func dropBits(stateData *State, bits uint8) {
	if bits > 32 {
		log.Fatal("Tried to drop more than 32 bits.")
	}

	if uint32(bits) > stateData.Bits {
		log.Fatal("Tried to drop more bits than we have.")
	}

	if bits == 32 {
		stateData.Head = stateData.Buffer
		stateData.Buffer = 0
	} else {
		stateData.Head = (stateData.Head << bits) | (stateData.Buffer >> (32 - bits))
		stateData.Buffer <<= bits
	}

	stateData.Bits -= uint32(bits)
}

// readBits reads a specified number of bits
func readBits(state *State, bits uint8) uint32 {
	return (state.Head >> (32 - bits))
}

// readCode reads a code from the Huffman tree
func readCode(huffmanTree *HuffmanTree, stateData *State, ioCode *uint16) {
	if huffmanTree.CompressedCodes[0] == 0 {
		log.Fatal("Trying to read code from an empty HuffmanTree.")
	}

	needBits(stateData, 32)
	tempIndex := uint16(0)
	bitsRead := readBits(stateData, 32)

	for bitsRead < huffmanTree.CompressedCodes[tempIndex] {
		tempIndex++
	}

	tempBits := huffmanTree.BitsLength[tempIndex]
	*ioCode = huffmanTree.SymbolValues[huffmanTree.SymbolValueOffset[tempIndex]-uint16(((bitsRead-huffmanTree.CompressedCodes[tempIndex])>>(32-tempBits)))]
	dropBits(stateData, tempBits)
}

// createHuffmanTree builds the Huffman tree
func createHuffmanTree(ioHuffmanTree *HuffmanTree, ioWorkingBitTab *[MAX_CODE_BITS_LENGTH]int16, ioWorkingCodeTab *[MAX_SYMBOL_VALUE]int16) {
	tempCode := uint32(0)
	tempBits := uint8(0)
	comparisonCodeIndex := uint16(0)
	symbolOffset := uint16(0)

	for tempBits < MAX_CODE_BITS_LENGTH {
		if (*ioWorkingBitTab)[tempBits] != -1 {
			tempSymbol := (*ioWorkingBitTab)[tempBits]
			for tempSymbol != -1 {
				// Registering the code
				ioHuffmanTree.SymbolValues[symbolOffset] = uint16(tempSymbol)
				symbolOffset++
				tempSymbol = (*ioWorkingCodeTab)[tempSymbol]
				tempCode-- // Decrement code value for next symbol
			}

			// Minimum code value for tempBits bits
			ioHuffmanTree.CompressedCodes[comparisonCodeIndex] = (tempCode + 1) << (32 - tempBits)

			// Number of bits for l_codeCompIndex index
			ioHuffmanTree.BitsLength[comparisonCodeIndex] = tempBits

			// Offset in symbol_values table to reach the value
			ioHuffmanTree.SymbolValueOffset[comparisonCodeIndex] = symbolOffset - 1

			comparisonCodeIndex++
		}
		tempCode = (tempCode << 1) + 1 // Increment code for next length
		tempBits++
	}
}

// fillTabsHelper updates the working bit and code tables based on the provided bits and symbol.
func fillTabsHelper(bits uint8, symbol int16, ioWorkingBitTab *[MAX_CODE_BITS_LENGTH]int16, ioWorkingCodeTab *[MAX_SYMBOL_VALUE]int16) {
	// Check for out of bounds
	if bits >= MAX_CODE_BITS_LENGTH {
		fmt.Fprintln(os.Stderr, "Error: Too many bits.")
		return // Exit the function to prevent further execution
	}

	if symbol >= MAX_SYMBOL_VALUE {
		fmt.Fprintln(os.Stderr, "Error: Too high symbol.")
		return // Exit the function to prevent further execution
	}

	if (*ioWorkingBitTab)[bits] == -1 {
		(*ioWorkingBitTab)[bits] = symbol
	} else {
		(*ioWorkingCodeTab)[symbol] = (*ioWorkingBitTab)[bits]
		(*ioWorkingBitTab)[bits] = symbol
	}
}

func initializeHuffmanTreeDict() {
	var workingBits [MAX_CODE_BITS_LENGTH]int16
	var workingCode [MAX_SYMBOL_VALUE]int16

	// Initialize working tables
	for i := range workingBits {
		workingBits[i] = -1 // Use -1 to indicate uninitialized
	}
	for i := range workingCode {
		workingCode[i] = -1 // Use -1 to indicate uninitialized
	}

	// Define your bits and symbols arrays
	bits := []uint8{
		3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6, 6, 7, 7, 7, 7, 7, 7, 7, 8, 8, 8, 8,
		8, 8, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
		10, 10, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 12, 12, 12, 12, 12, 12, 12, 13,
		13, 13, 13, 13, 13, 14, 14, 14, 14, 15, 15, 15, 15, 15, 15, 15, 15, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
	} // Example values, adjust as needed
	symbols := []int16{
		0x0A, 0x09, 0x08, 0x0C, 0x0B, 0x07, 0x00, 0xE0, 0x2A, 0x29, 0x06, 0x4A, 0x40, 0x2C, 0x2B,
		0x28, 0x20, 0x05, 0x04, 0x49, 0x48, 0x27, 0x26, 0x25, 0x0D, 0x03, 0x6A, 0x69, 0x4C, 0x4B,
		0x47, 0x24, 0xE8, 0xA0, 0x89, 0x88, 0x68, 0x67, 0x63, 0x60, 0x46, 0x23, 0xE9, 0xC9, 0xC0,
		0xA9, 0xA8, 0x8A, 0x87, 0x80, 0x66, 0x65, 0x45, 0x44, 0x43, 0x2D, 0x02, 0x01, 0xE5, 0xC8,
		0xAA, 0xA5, 0xA4, 0x8B, 0x85, 0x84, 0x6C, 0x6B, 0x64, 0x4D, 0x0E, 0xE7, 0xCA, 0xC7, 0xA7,
		0xA6, 0x86, 0x83, 0xE6, 0xE4, 0xC4, 0x8C, 0x2E, 0x22, 0xEC, 0xC6, 0x6D, 0x4E, 0xEA, 0xCC,
		0xAC, 0xAB, 0x8D, 0x11, 0x10, 0x0F, 0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7,
		0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0, 0xEF, 0xEE, 0xED, 0xEB, 0xE3, 0xE2, 0xE1, 0xDF,
		0xDE, 0xDD, 0xDC, 0xDB, 0xDA, 0xD9, 0xD8, 0xD7, 0xD6, 0xD5, 0xD4, 0xD3, 0xD2, 0xD1, 0xD0,
		0xCF, 0xCE, 0xCD, 0xCB, 0xC5, 0xC3, 0xC2, 0xC1, 0xBF, 0xBE, 0xBD, 0xBC, 0xBB, 0xBA, 0xB9,
		0xB8, 0xB7, 0xB6, 0xB5, 0xB4, 0xB3, 0xB2, 0xB1, 0xB0, 0xAF, 0xAE, 0xAD, 0xA3, 0xA2, 0xA1,
		0x9F, 0x9E, 0x9D, 0x9C, 0x9B, 0x9A, 0x99, 0x98, 0x97, 0x96, 0x95, 0x94, 0x93, 0x92, 0x91,
		0x90, 0x8F, 0x8E, 0x82, 0x81, 0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78, 0x77, 0x76,
		0x75, 0x74, 0x73, 0x72, 0x71, 0x70, 0x6F, 0x6E, 0x62, 0x61, 0x5F, 0x5E, 0x5D, 0x5C, 0x5B,
		0x5A, 0x59, 0x58, 0x57, 0x56, 0x55, 0x54, 0x53, 0x52, 0x51, 0x50, 0x4F, 0x42, 0x41, 0x3F,
		0x3E, 0x3D, 0x3C, 0x3B, 0x3A, 0x39, 0x38, 0x37, 0x36, 0x35, 0x34, 0x33, 0x32, 0x31, 0x30,
		0x2F, 0x21, 0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18, 0x17, 0x16, 0x15, 0x14, 0x13,
		0x12} // Example values, adjust as needed

	tempSymbol := len(bits) // Calculate number of symbols

	// Populate the working tables
	for i := 0; i < tempSymbol; i++ {
		fillTabsHelper(bits[i], symbols[i], &workingBits, &workingCode)
	}

	// Build the Huffman tree
	createHuffmanTree(&huffmanTreeDict, &workingBits, &workingCode)
}

// Function to parse the Huffman tree
func parseHuffmanTree(stateData *State, ioHuffmanTree *HuffmanTree) {
	// Reading the number of symbols to read
	needBits(stateData, 16)
	numberSymbolData := uint16(readBits(stateData, 16)) // C-style cast equivalent
	dropBits(stateData, 16)

	if numberSymbolData > MAX_SYMBOL_VALUE {
		fmt.Fprintln(os.Stderr, "Too many symbols to decode.")
	}

	var workingBits [MAX_CODE_BITS_LENGTH]int16
	var workingCode [MAX_SYMBOL_VALUE]int16

	// Initialize our workingBits and workingCode
	for i := range workingBits {
		workingBits[i] = -1 // Using -1 to indicate uninitialized
	}
	for i := range workingCode {
		workingCode[i] = -1 // Using -1 to indicate uninitialized
	}
	var numberSymbol = int16(numberSymbolData)
	remainingSymbol := numberSymbol - 1

	// Fetching the code repartition
	for remainingSymbol >= 0 {
		var tempCode uint16
		readCode(&huffmanTreeDict, stateData, &tempCode)

		codeNumberBits := tempCode & 0x1F
		codeNumberSymbol := int16((tempCode >> 5) + 1)

		if codeNumberBits == 0 {
			remainingSymbol -= codeNumberSymbol
		} else {
			for codeNumberSymbol > 0 {
				if workingBits[codeNumberBits] == -1 {
					workingBits[codeNumberBits] = int16(remainingSymbol)
				} else {
					workingCode[remainingSymbol] = workingBits[codeNumberBits]
					workingBits[codeNumberBits] = int16(remainingSymbol)
				}
				remainingSymbol--
				codeNumberSymbol--
			}
		}
	}

	// Effectively build the Huffman tree
	createHuffmanTree(ioHuffmanTree, &workingBits, &workingCode)
}
func inflateData(stateData *State, outputBuffer *[]uint8, outputBufferSize uint32) {
	tempOutputPosition := uint32(0)

	// Reading the constant write size addition value
	needBits(stateData, 8)
	dropBits(stateData, 4)
	writeSizeConstantAddition := (readBits(stateData, 4) + 1)
	dropBits(stateData, 4)

	// Declaring our Huffman Trees
	var huffmanTreeSymbol, huffmanTreeCopy HuffmanTree

	for tempOutputPosition < outputBufferSize {
		// Resetting Huffman trees
		huffmanTreeSymbol = HuffmanTree{}
		huffmanTreeCopy = HuffmanTree{}

		// Reading Huffman Trees
		parseHuffmanTree(stateData, &huffmanTreeSymbol)
		parseHuffmanTree(stateData, &huffmanTreeCopy)

		// Reading MaxCount
		needBits(stateData, 4)
		maxCount := (readBits(stateData, 4) + 1) << 12
		dropBits(stateData, 4)

		currentCodeReadCount := uint32(0)

		for currentCodeReadCount < maxCount && tempOutputPosition < outputBufferSize {
			currentCodeReadCount++

			// Reading next code
			var tempCode uint16
			readCode(&huffmanTreeSymbol, stateData, &tempCode)

			if tempCode < 0x100 {
				(*outputBuffer)[tempOutputPosition] = uint8(tempCode) // Cast to uint8
				tempOutputPosition++
				continue
			}

			// We are in copy mode!
			// Reading the additional info to know the write size
			tempCode -= 0x100

			// Write size
			codeDivision4 := tempCode / 4
			rem := tempCode % 4

			var writeSize uint32
			switch {
			case codeDivision4 == 0:
				writeSize = uint32(tempCode)
			case codeDivision4 < 7:
				writeSize = uint32((1 << (codeDivision4 - 1)) * (4 + rem))
			case tempCode == 28:
				writeSize = 0xFF
			default:
				fmt.Fprintln(os.Stderr, "Invalid value for writeSize code.")
				os.Exit(1)
			}

			// Additional bits
			if codeDivision4 > 1 && tempCode != 28 {
				writeSizeAddition := codeDivision4 - 1
				needBits(stateData, uint8(writeSizeAddition))
				writeSize |= readBits(stateData, uint8(writeSizeAddition))
				dropBits(stateData, uint8(writeSizeAddition))
			}
			writeSize += writeSizeConstantAddition

			// Write offset
			// Reading the write offset
			readCode(&huffmanTreeCopy, stateData, &tempCode)

			codeDivision2 := tempCode / 2

			var writeOffset uint32
			switch {
			case codeDivision2 == 0:
				writeOffset = uint32(tempCode)
			case codeDivision2 < 17:
				writeOffset = uint32((1 << (codeDivision2 - 1)) * (2 + (tempCode % 2)))
			default:
				fmt.Fprintln(os.Stderr, "Invalid value for writeOffset code.")
				os.Exit(1)
			}

			// Additional bits
			if codeDivision2 > 1 {
				writeOffsetAdditionBits := codeDivision2 - 1
				needBits(stateData, uint8(writeOffsetAdditionBits))
				writeOffset |= readBits(stateData, uint8(writeOffsetAdditionBits))
				dropBits(stateData, uint8(writeOffsetAdditionBits))
			}
			writeOffset += 1

			alreadyWritten := uint32(0)
			for alreadyWritten < writeSize && tempOutputPosition < outputBufferSize {
				(*outputBuffer)[tempOutputPosition] = (*outputBuffer)[tempOutputPosition-writeOffset]
				tempOutputPosition++
				alreadyWritten++
			}
		}
	}
}

// Convert uint8 buffer to uint32 buffer
func convertU8ToU32(input []uint8) ([]uint32, error) {
	inputSize := len(input)
	if inputSize%4 != 0 {
		return nil, errors.New("input size is not a multiple of 4")
	}

	outputSize := inputSize / 4
	output := make([]uint32, outputSize)

	for i := 0; i < outputSize; i++ {
		output[i] = uint32(input[i*4]) |
			uint32(input[i*4+1])<<8 |
			uint32(input[i*4+2])<<16 |
			uint32(input[i*4+3])<<24 // Little-endian conversion
	}

	return output, nil
}

// Inflate the buffer
func inflateBuffer(inputBufferSize uint32, inputBuffer []uint8, outputBufferSize *uint32, customOutputBufferSize uint32) ([]uint8, error) {
	if inputBuffer == nil {
		return nil, errors.New("input buffer is null")
	}

	if !huffmanTreeDictInitialized {
		initializeHuffmanTreeDict()
		huffmanTreeDictInitialized = true
	}

	if huffmanTreeDict.CompressedCodes[0] == 0 {
		return nil, errors.New("huffman tree empty")
	}

	// Convert uint8 buffer to uint32 buffer
	u32InputBuffer, err := convertU8ToU32(inputBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input buffer: %v", err)
	}

	log.Println("Initialize state!")

	// Initialize state
	stateData := &State{
		u32InputBuffer,
		uint32(len(u32InputBuffer)),
		0,
		0,
		0,
		0,
		false,
	}

	// Skipping header & getting size of the uncompressed data
	needBits(stateData, 32)
	dropBits(stateData, 32)

	// Getting size of the uncompressed data
	needBits(stateData, 32)
	tempOutputBufferSize := readBits(stateData, 32)
	dropBits(stateData, 32)

	if *outputBufferSize != 0 {
		// We do not take max here as we won't be able to have more than the output available
		if tempOutputBufferSize > *outputBufferSize {
			tempOutputBufferSize = *outputBufferSize
		}
	}

	*outputBufferSize = tempOutputBufferSize

	if customOutputBufferSize > 0 {
		tempOutputBufferSize = customOutputBufferSize
	}

	// Allocate memory for output buffer
	outputBuffer := make([]uint8, tempOutputBufferSize)

	// Inflate data
	inflateData(stateData, &outputBuffer, tempOutputBufferSize)

	return outputBuffer, nil
}
