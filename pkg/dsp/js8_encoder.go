package dsp

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// JS8 alphabet for 6-bit encoding (64 characters)
const js8Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-+"

// Costas arrays for JS8 Normal mode
var costasNormal = [3][7]int{
	{4, 2, 5, 6, 1, 3, 0}, // Start
	{4, 2, 5, 6, 1, 3, 0}, // Middle
	{4, 2, 5, 6, 1, 3, 0}, // End
}

// Alphabet lookup table (initialized once)
var alphabetTable [256]uint8
var alphabetTableInit = false

// CRC-12 lookup table (polynomial 0xc06)
var crc12Table [256]uint16
var crc12TableInit = false

// Initialize alphabet lookup table
func initAlphabetTable() {
	if alphabetTableInit {
		return
	}

	// Initialize all entries as invalid
	for i := 0; i < 256; i++ {
		alphabetTable[i] = 0xff
	}

	// Set valid alphabet characters
	for i, c := range js8Alphabet {
		alphabetTable[uint8(c)] = uint8(i)
	}
	alphabetTableInit = true
}

// Convert character to 6-bit word
func alphabetWord(c byte) (uint8, error) {
	initAlphabetTable()

	word := alphabetTable[c]
	if word == 0xff {
		return 0, fmt.Errorf("invalid character '%c' in JS8 message", c)
	}
	return word, nil
}

// Initialize CRC-12 lookup table
func initCRC12Table() {
	if crc12TableInit {
		return
	}

	const poly = 0xc06
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 4 // CRC-12, so shift by 4
		for j := 0; j < 8; j++ {
			if crc&0x800 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
			crc &= 0xfff // Keep only 12 bits
		}
		crc12Table[i] = crc
	}
	crc12TableInit = true
}

// Compute CRC-12 of byte slice
func computeCRC12(data []byte) uint16 {
	initCRC12Table()

	var crc uint16 = 0
	for _, b := range data {
		tblIdx := ((crc >> 4) ^ uint16(b)) & 0xff
		crc = ((crc << 8) ^ crc12Table[tblIdx]) & 0xfff
	}
	return crc ^ 42 // XOR with 42 as in original
}

// Placeholder parity matrix function
// TODO: Replace with real 87x87 parity matrix from JS8Call
func getParityBit(row, col int) bool {
	// Placeholder pattern - this generates incorrect parity but allows testing
	return ((row*13 + col*17) % 3) == 0
}

// JS8Encoder represents a pure Go JS8 encoder
type JS8Encoder struct{}

// NewJS8Encoder creates a new JS8 encoder
func NewJS8Encoder() *JS8Encoder {
	return &JS8Encoder{}
}

// EncodeMessage encodes a JS8 message to tone sequence
func (e *JS8Encoder) EncodeMessage(message string, frameType int) ([]int, error) {
	// Validate message length
	if len(message) != 12 {
		return nil, fmt.Errorf("message must be exactly 12 characters, got %d", len(message))
	}

	// Create 11-byte array for the 87-bit message
	bytes := make([]byte, 11)

	// Pack 12 characters into 9 bytes (72 bits)
	for i, j := 0, 0; i < 12; i, j = i+4, j+3 {
		// Convert 4 characters to 6-bit words
		word0, err := alphabetWord(message[i])
		if err != nil {
			return nil, fmt.Errorf("character %d: %w", i, err)
		}
		word1, err := alphabetWord(message[i+1])
		if err != nil {
			return nil, fmt.Errorf("character %d: %w", i+1, err)
		}
		word2, err := alphabetWord(message[i+2])
		if err != nil {
			return nil, fmt.Errorf("character %d: %w", i+2, err)
		}
		word3, err := alphabetWord(message[i+3])
		if err != nil {
			return nil, fmt.Errorf("character %d: %w", i+3, err)
		}

		// Pack 4 6-bit words into 3 bytes (24 bits)
		words := (uint32(word0) << 18) |
			(uint32(word1) << 12) |
			(uint32(word2) << 6) |
			uint32(word3)

		bytes[j] = byte(words >> 16)
		bytes[j+1] = byte(words >> 8)
		bytes[j+2] = byte(words)
	}

	// Add frame type (3 bits)
	bytes[9] = byte(frameType&0x07) << 5

	// Compute and add CRC-12
	crc := computeCRC12(bytes)
	bytes[9] |= byte((crc >> 7) & 0x1F)
	bytes[10] = byte((crc & 0x7F) << 1)

	// Initialize tone array (79 tones total)
	tones := make([]int, 79)

	// Add Costas arrays at positions 0, 36, and 72
	costasPositions := []int{0, 36, 72}
	for i, pos := range costasPositions {
		copy(tones[pos:pos+7], costasNormal[i][:])
	}

	// Generate parity and output data (29 3-bit words each)
	parityData := tones[7:36]  // 29 parity tones
	outputData := tones[43:72] // 29 output tones

	var outputBits int
	var outputByte int
	var outputMask byte = 0x80
	var outputWord, parityWord byte

	for i := 0; i < 87; i++ {
		// Compute parity bit for position i
		var parityBits int
		var parityByte int
		var parityMask byte = 0x80

		for j := 0; j < 87; j++ {
			if getParityBit(i, j) && (bytes[parityByte]&parityMask) != 0 {
				parityBits++
			}
			if parityMask == 1 {
				parityByte++
				parityMask = 0x80
			} else {
				parityMask >>= 1
			}
		}

		// Accumulate bits
		parityWord = (parityWord << 1) | byte(parityBits&1)
		if (bytes[outputByte] & outputMask) != 0 {
			outputWord = (outputWord << 1) | 1
		} else {
			outputWord = outputWord << 1
		}

		if outputMask == 1 {
			outputByte++
			outputMask = 0x80
		} else {
			outputMask >>= 1
		}

		// Output 3-bit words
		outputBits++
		if outputBits == 3 {
			parityData[(i-2)/3] = int(parityWord)
			outputData[(i-2)/3] = int(outputWord)
			parityWord = 0
			outputWord = 0
			outputBits = 0
		}
	}

	return tones, nil
}

// GenerateAudio converts tone sequence to audio samples
func (e *JS8Encoder) GenerateAudio(tones []int, sampleRate int) []int16 {
	// JS8 Normal mode parameters
	const duration = 15.0                       // seconds
	const baseFreq = 1000.0                     // Hz
	freqSpacing := float64(sampleRate) / 2048.0 // ~5.86 Hz for 12kHz

	toneDuration := duration / float64(len(tones))
	samplesPerTone := int(toneDuration * float64(sampleRate))
	totalSamples := len(tones) * samplesPerTone

	audio := make([]int16, totalSamples)
	sampleIdx := 0

	for _, tone := range tones {
		freq := baseFreq + float64(tone)*freqSpacing
		omega := 2.0 * math.Pi * freq / float64(sampleRate)

		for i := 0; i < samplesPerTone && sampleIdx < totalSamples; i++ {
			// Generate sine wave
			amplitude := 16384.0 // ~50% of int16 range
			sample := amplitude * math.Sin(omega*float64(i))
			audio[sampleIdx] = int16(sample)
			sampleIdx++
		}
	}

	return audio
}

// EncodeToAudio is a convenience function that encodes message directly to audio
func (e *JS8Encoder) EncodeToAudio(message string, frameType int, sampleRate int) ([]int16, error) {
	tones, err := e.EncodeMessage(message, frameType)
	if err != nil {
		return nil, err
	}

	audio := e.GenerateAudio(tones, sampleRate)
	return audio, nil
}

// ValidateMessage checks if a message contains only valid JS8 characters
func ValidateMessage(message string) error {
	initAlphabetTable()

	for i, c := range []byte(message) {
		if alphabetTable[c] == 0xff {
			return fmt.Errorf("invalid character '%c' at position %d", c, i)
		}
	}
	return nil
}

// PreprocessJS8Message preprocesses a message to make it compatible with JS8 encoding
// This handles common JS8 message formats and removes invalid characters
func PreprocessJS8Message(message string) string {
	// Handle common JS8 message patterns
	message = strings.TrimSpace(message)

	// Convert to uppercase for JS8 compatibility
	message = strings.ToUpper(message)

	// Handle CQ messages - replace spaces with standard JS8 format
	if strings.HasPrefix(message, "CQ ") {
		// "CQ CQ DE W1AW W1AW K" -> "CQ-DE-W1AW-K" (simplified for JS8 Normal mode)
		parts := strings.Fields(message)
		if len(parts) >= 4 && parts[0] == "CQ" && parts[2] == "DE" {
			// Extract callsign (parts[3])
			callsign := parts[3]
			return fmt.Sprintf("CQ-DE-%s-K", callsign)
		}
		// Generic CQ format
		return "CQ-CQ-CQ"
	}

	// Handle heartbeat messages
	if strings.Contains(message, "HEARTBEAT") || strings.HasPrefix(message, "HB") {
		// Extract callsign and grid if present
		parts := strings.Fields(message)
		var callsign, grid string

		// Find callsign and grid separately
		for _, part := range parts {
			cleanPart := strings.Trim(part, ":,.")
			if len(cleanPart) >= 3 && ContainsLettersAndNumbers(cleanPart) && !strings.Contains(cleanPart, "HEARTBEAT") && callsign == "" {
				callsign = cleanPart
			} else if IsGridSquare(cleanPart) && grid == "" {
				grid = cleanPart
			}
		}

		// Format as JS8 heartbeat (keep within 12 character limit)
		if callsign != "" {
			if grid != "" {
				// Use compact format: HB + callsign + grid (truncate if too long)
				compact := fmt.Sprintf("HB%s%s", callsign, grid)
				if len(compact) <= 12 {
					return compact
				}
				// If too long, use grid without full callsign or truncate
				if len(callsign) > 6 {
					callsign = callsign[:6] // Truncate long callsigns
				}
				compact = fmt.Sprintf("HB%s", callsign)
				if len(compact) <= 12 {
					return compact
				}
			}
			// Just callsign heartbeat
			if len(callsign) <= 8 { // HB + 8 chars = 10 chars, room for padding
				return fmt.Sprintf("HB%s", callsign)
			} else {
				return fmt.Sprintf("HB%s", callsign[:8])
			}
		}
		return "HBAUTO"
	}

	// For other messages, remove spaces and invalid characters
	var result strings.Builder
	for _, c := range message {
		// Check if character is in JS8 alphabet
		if c < 256 {
			initAlphabetTable()
			if alphabetTable[byte(c)] != 0xff {
				result.WriteRune(c)
			}
		}
	}

	return result.String()
}

// ContainsLettersAndNumbers checks if a string contains both letters and numbers (exported for testing)
func ContainsLettersAndNumbers(s string) bool {
	hasLetter := false
	hasNumber := false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasNumber = true
		}
		if hasLetter && hasNumber {
			return true
		}
	}
	return hasLetter && hasNumber
}

// IsGridSquare checks if a string looks like a Maidenhead grid square (exported for testing)
func IsGridSquare(s string) bool {
	if len(s) < 4 || len(s) > 6 {
		return false
	}
	// Basic pattern: AA00 or AA00aa
	return len(s) >= 4 &&
		s[0] >= 'A' && s[0] <= 'R' &&
		s[1] >= 'A' && s[1] <= 'R' &&
		s[2] >= '0' && s[2] <= '9' &&
		s[3] >= '0' && s[3] <= '9'
}

// PadMessage pads a message to exactly 12 characters using a fill character
func PadMessage(message string, fillChar byte) (string, error) {
	if len(message) > 12 {
		return "", errors.New("message too long (max 12 characters)")
	}

	// Validate fill character
	if err := ValidateMessage(string(fillChar)); err != nil {
		return "", fmt.Errorf("invalid fill character: %w", err)
	}

	// Pad with fill character
	padded := message
	for len(padded) < 12 {
		padded += string(fillChar)
	}

	return padded, nil
}
