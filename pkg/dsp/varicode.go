package dsp

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
)

// JS8 Frame types
type FrameType uint8

const (
	FrameUnknown          FrameType = 255 // Sentinel value
	FrameHeartbeat        FrameType = 0   // [000]
	FrameCompound         FrameType = 1   // [001]
	FrameCompoundDirected FrameType = 2   // [010]
	FrameDirected         FrameType = 3   // [011]
	FrameData             FrameType = 4   // [10X]
	FrameDataCompressed   FrameType = 6   // [11X]
)

const FrameTypeMax = 6

// Frame type strings for debugging
var frameTypeStrings = map[FrameType]string{
	FrameHeartbeat:        "FrameHeartbeat",
	FrameCompound:         "FrameCompound",
	FrameCompoundDirected: "FrameCompoundDirected",
	FrameDirected:         "FrameDirected",
	FrameData:             "FrameData",
	FrameDataCompressed:   "FrameDataCompressed",
	FrameUnknown:          "FrameUnknown",
}

func (f FrameType) String() string {
	if str, ok := frameTypeStrings[f]; ok {
		return str
	}
	return "FrameUnknown"
}

// JS8 transmission types
type TransmissionType uint8

const (
	JS8Call      TransmissionType = 0 // Any other frame of the message
	JS8CallFirst TransmissionType = 1 // First frame of a message
	JS8CallLast  TransmissionType = 2 // Last frame of a message
	JS8CallData  TransmissionType = 4 // Flagged frame (no frame type header)
)

// Alphabets for encoding
const (
	// Base-41 alphabet for FT8 freetext transmission
	alphabet41 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ+-./?)"
	nalphabet  = 41

	// Base-64 alphabet for 72-bit encoding
	alphabet72 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-+/?."

	// Alphanumeric alphabet for callsigns and grids
	alphanumeric = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ /@"
)

// Directed commands mapping
var directedCmds = map[string]int{
	" HEARTBEAT":   -1,
	" HB":          -1,
	" CQ":          -1,
	" SNR?":        0,
	"?":            0,
	" DIT DIT":     1,
	" HEARING?":    3,
	" GRID?":       4,
	">":            5,
	" STATUS?":     6,
	" STATUS":      7,
	" HEARING":     8,
	" MSG":         9,
	" MSG TO:":     10,
	" QUERY":       11,
	" QUERY MSGS":  12,
	" QUERY MSGS?": 12,
	" QUERY CALL":  13,
	" GRID":        15,
	" INFO?":       16,
	" INFO":        17,
	" FB":          18,
	" HW CPY?":     19,
	" SK":          20,
	" RR":          21,
	" QSL?":        22,
	" QSL":         23,
	" CMD":         24,
	" SNR":         25,
	" NO":          26,
	" YES":         27,
}

// Regular expressions for parsing
var (
	gridPattern     = regexp.MustCompile(`[A-X]{2}[0-9]{2}(?:[A-X]{2}(?:[0-9]{2})?)?`)
	callsignPattern = regexp.MustCompile(`[A-Z0-9]{1,3}[0-9][A-Z]{1,3}(?:/[A-Z0-9]+)?`)
)

// Bit manipulation utilities
func intToBits(value uint64, bitCount int) []bool {
	bits := make([]bool, bitCount)
	for i := 0; i < bitCount; i++ {
		bits[bitCount-1-i] = (value>>i)&1 == 1
	}
	return bits
}

func bitsToInt(bits []bool) uint64 {
	var value uint64
	for i, bit := range bits {
		if bit {
			value |= 1 << (len(bits) - 1 - i)
		}
	}
	return value
}

// Pack/unpack functions for base-41 encoding

// Unpack5bits extracts a 5-bit value from a character
func Unpack5bits(value string) uint8 {
	if len(value) == 0 {
		return 0
	}
	idx := strings.IndexRune(alphabet41, rune(value[0]))
	if idx == -1 {
		return 0
	}
	return uint8(idx)
}

// Pack5bits packs a 5-bit value (0-31) into a single character
func Pack5bits(packed uint8) string {
	return string(alphabet41[packed%32])
}

// Unpack6bits extracts a 6-bit value from a character
func Unpack6bits(value string) uint8 {
	if len(value) == 0 {
		return 0
	}
	idx := strings.IndexRune(alphabet41, rune(value[0]))
	if idx == -1 {
		return 0
	}
	return uint8(idx)
}

// Pack6bits packs a 6-bit value (0-40) into a single character
func Pack6bits(packed uint8) string {
	return string(alphabet41[packed%41])
}

// Unpack16bits extracts a 16-bit value from 3 base-41 characters
func Unpack16bits(value string) uint16 {
	if len(value) < 3 {
		return 0
	}

	a := strings.IndexRune(alphabet41, rune(value[0]))
	b := strings.IndexRune(alphabet41, rune(value[1]))
	c := strings.IndexRune(alphabet41, rune(value[2]))

	if a == -1 || b == -1 || c == -1 {
		return 0
	}

	unpacked := (nalphabet*nalphabet)*a + nalphabet*b + c
	if unpacked > (1<<16)-1 {
		return 0 // BASE-41 can produce values larger than 16 bits
	}

	return uint16(unpacked)
}

// Pack16bits packs a 16-bit value into 3 base-41 characters
func Pack16bits(packed uint16) string {
	var out strings.Builder

	tmp := int(packed) / (nalphabet * nalphabet)
	out.WriteByte(alphabet41[tmp])

	tmp = (int(packed) - (tmp * (nalphabet * nalphabet))) / nalphabet
	out.WriteByte(alphabet41[tmp])

	tmp = int(packed) % nalphabet
	out.WriteByte(alphabet41[tmp])

	return out.String()
}

// Unpack32bits extracts a 32-bit value from 6 base-41 characters
func Unpack32bits(value string) uint32 {
	if len(value) < 6 {
		return 0
	}
	return uint32(Unpack16bits(value[:3]))<<16 | uint32(Unpack16bits(value[3:6]))
}

// Pack32bits packs a 32-bit value into 6 base-41 characters
func Pack32bits(packed uint32) string {
	a := uint16((packed & 0xFFFF0000) >> 16)
	b := uint16(packed & 0xFFFF)
	return Pack16bits(a) + Pack16bits(b)
}

// Unpack64bits extracts a 64-bit value from 12 base-41 characters
func Unpack64bits(value string) uint64 {
	if len(value) < 12 {
		return 0
	}
	return uint64(Unpack32bits(value[:6]))<<32 | uint64(Unpack32bits(value[6:12]))
}

// Pack64bits packs a 64-bit value into 12 base-41 characters
func Pack64bits(packed uint64) string {
	a := uint32((packed & 0xFFFFFFFF00000000) >> 32)
	b := uint32(packed & 0xFFFFFFFF)
	return Pack32bits(a) + Pack32bits(b)
}

// Unpack72bits extracts 72 bits from 12 base-64 characters, returning first 64 bits and last 8 bits
func Unpack72bits(text string, pRem *uint8) uint64 {
	if len(text) < 12 {
		return 0
	}

	var value uint64
	var rem uint8

	for i := 0; i < 10; i++ {
		idx := strings.IndexRune(alphabet72, rune(text[i]))
		if idx == -1 {
			idx = 0
		}
		value |= uint64(idx) << (58 - 6*i)
	}

	remHigh := strings.IndexRune(alphabet72, rune(text[10]))
	if remHigh == -1 {
		remHigh = 0
	}
	value |= uint64(remHigh >> 2)

	remLow := strings.IndexRune(alphabet72, rune(text[11]))
	if remLow == -1 {
		remLow = 0
	}
	rem = uint8((remHigh&3)<<6) | uint8(remLow&63)

	if pRem != nil {
		*pRem = rem
	}

	return value
}

// Pack72bits packs 72 bits (64 + 8) into 12 base-64 characters
func Pack72bits(value uint64, rem uint8) string {
	packed := make([]byte, 12)

	remHigh := uint8((value&15)<<2) | (rem >> 6)
	remLow := rem & 63
	value = value >> 4

	packed[11] = alphabet72[remLow]
	packed[10] = alphabet72[remHigh]

	for i := 9; i >= 0; i-- {
		packed[i] = alphabet72[value&63]
		value >>= 6
	}

	return string(packed)
}

// Checksum functions

// Checksum16 computes a 16-bit CRC checksum and packs it into 3 characters
func Checksum16(input string) string {
	data := []byte(input)
	crc := crc32.ChecksumIEEE(data) & 0xFFFF // Use only lower 16 bits
	checksum := Pack16bits(uint16(crc))

	// Pad to 3 characters if needed
	for len(checksum) < 3 {
		checksum += " "
	}
	return checksum
}

// Checksum16Valid validates a 16-bit CRC checksum
func Checksum16Valid(checksum, input string) bool {
	return Checksum16(input) == checksum
}

// Checksum32 computes a 32-bit CRC checksum and packs it into 6 characters
func Checksum32(input string) string {
	data := []byte(input)
	crc := crc32.ChecksumIEEE(data)
	checksum := Pack32bits(crc)

	// Pad to 6 characters if needed
	for len(checksum) < 6 {
		checksum += " "
	}
	return checksum
}

// Checksum32Valid validates a 32-bit CRC checksum
func Checksum32Valid(checksum, input string) bool {
	return Checksum32(input) == checksum
}

// String utility functions

// Rstrip removes trailing whitespace
func Rstrip(str string) string {
	return strings.TrimRight(str, " \t\n\r")
}

// Lstrip removes leading whitespace
func Lstrip(str string) string {
	return strings.TrimLeft(str, " \t\n\r")
}

// Escape escapes special characters in text
func Escape(text string) string {
	// Simple escape implementation - can be enhanced as needed
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\r", "\\r")
	text = strings.ReplaceAll(text, "\t", "\\t")
	return text
}

// Unescape unescapes special characters in text
func Unescape(text string) string {
	// Simple unescape implementation - can be enhanced as needed
	text = strings.ReplaceAll(text, "\\t", "\t")
	text = strings.ReplaceAll(text, "\\r", "\r")
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}

// Parsing functions

// ParseCallsigns extracts callsigns from input text
func ParseCallsigns(input string) []string {
	matches := callsignPattern.FindAllString(input, -1)
	return matches
}

// ParseGrids extracts grid squares from input text
func ParseGrids(input string) []string {
	matches := gridPattern.FindAllString(input, -1)
	return matches
}

// Command validation functions

// IsSNRCommand checks if a command is SNR-related
func IsSNRCommand(cmd string) bool {
	return cmd == " SNR?" || cmd == " SNR"
}

// IsCommandAllowed checks if a command is allowed
func IsCommandAllowed(cmd string) bool {
	_, exists := directedCmds[cmd]
	return exists
}

// IsCommandBuffered checks if a command should be buffered
func IsCommandBuffered(cmd string) bool {
	// Commands that should be buffered for batch processing
	bufferedCmds := map[string]bool{
		" MSG":         true,
		" MSG TO:":     true,
		" QUERY":       true,
		" QUERY MSGS":  true,
		" QUERY MSGS?": true,
		" INFO":        true,
		" STATUS":      true,
	}
	return bufferedCmds[cmd]
}

// IsCommandChecksumed returns checksum type for command (0=none, 16=16bit, 32=32bit)
func IsCommandChecksumed(cmd string) int {
	// Commands that require checksums
	checksumCmds := map[string]int{
		" MSG":     16,
		" MSG TO:": 16,
		" INFO":    16,
		" STATUS":  16,
		" QUERY":   32,
	}
	return checksumCmds[cmd]
}

// IsCommandAutoreply checks if a command should generate an automatic reply
func IsCommandAutoreply(cmd string) bool {
	autoreplyCmds := map[string]bool{
		" SNR?":        true,
		"?":            true,
		" GRID?":       true,
		" STATUS?":     true,
		" HEARING?":    true,
		" INFO?":       true,
		" QUERY MSGS":  true,
		" QUERY MSGS?": true,
	}
	return autoreplyCmds[cmd]
}

// Utility functions for JS8 specific formatting

// CQString formats a CQ string with number
func CQString(number int) string {
	if number <= 0 {
		return "CQ"
	}
	return fmt.Sprintf("CQ%d", number)
}

// HBString formats a heartbeat string with number
func HBString(number int) string {
	if number <= 0 {
		return "HB"
	}
	return fmt.Sprintf("HB%d", number)
}

// StartsWithCQ checks if text starts with CQ
func StartsWithCQ(text string) bool {
	return strings.HasPrefix(strings.ToUpper(text), "CQ")
}

// StartsWithHB checks if text starts with HB (heartbeat)
func StartsWithHB(text string) bool {
	return strings.HasPrefix(strings.ToUpper(text), "HB")
}

// FormatSNR formats SNR value for display
func FormatSNR(snr int) string {
	if snr >= 0 {
		return fmt.Sprintf("+%02d", snr)
	}
	return fmt.Sprintf("%03d", snr)
}

// FormatPWR formats power value in dBm for display
func FormatPWR(dbm int) string {
	return fmt.Sprintf("%ddBm", dbm)
}

// Grid conversion functions (simplified)

// Deg2Grid converts decimal degrees to Maidenhead grid square (matches C++ implementation)
func Deg2Grid(dlong, dlat float32) string {
	// Handle longitude wrapping
	if dlong < -180 {
		dlong += 360
	}
	if dlong > 180 {
		dlong -= 360
	}

	// JS8 deg2grid calculation (matches C++ code)
	nlong := int(60.0 * (180.0 - dlong) / 5)

	n1 := nlong / 240
	n2 := (nlong - 240*n1) / 24
	n3 := nlong - 240*n1 - 24*n2

	grid := make([]byte, 6)
	grid[0] = byte('A' + n1)
	grid[2] = byte('0' + n2)
	grid[4] = byte('a' + n3)

	nlat := int(60.0 * (dlat + 90) / 2.5)
	n1 = nlat / 240
	n2 = (nlat - 240*n1) / 24
	n3 = nlat - 240*n1 - 24*n2

	grid[1] = byte('A' + n1)
	grid[3] = byte('0' + n2)
	grid[5] = byte('a' + n3)

	return string(grid[:4]) // Return only the 4-character grid
}

// Grid2Deg converts Maidenhead grid square to decimal degrees (matches C++ implementation)
func Grid2Deg(grid string) (float32, float32) {
	if len(grid) < 4 {
		return 0, 0
	}

	// Pad grid to 6 characters if needed (add "mm" for subsquares)
	g := strings.ToUpper(grid)
	if len(g) < 6 {
		g = g[:4] + "mm"
	}

	// Make field/square uppercase, subsquare lowercase
	g = strings.ToUpper(g[:4]) + strings.ToLower(g[4:6])

	// JS8 grid2deg calculation (matches C++ code)
	nlong := 180 - 20*(int(g[0])-int('A'))
	n20d := 2 * (int(g[2]) - int('0'))
	xminlong := float32(5) * (float32(int(g[4])-int('a')) + 0.5)
	dlong := float32(nlong) - float32(n20d) - xminlong/60.0

	nlat := -90 + 10*(int(g[1])-int('A')) + (int(g[3]) - int('0'))
	xminlat := float32(2.5) * (float32(int(g[5])-int('a')) + 0.5)
	dlat := float32(nlat) + xminlat/60.0

	return dlong, dlat
}

// PackGrid packs a grid square into 16 bits using JS8 format
func PackGrid(grid string) uint16 {
	if len(grid) < 4 {
		return (1 << 15) - 1 // Return max value for invalid input
	}

	grid = strings.TrimSpace(strings.ToUpper(grid[:4]))
	dlong, dlat := Grid2Deg(grid)

	ilong := int(dlong)
	ilat := int(dlat + 90)

	return uint16(((ilong+180)/2)*180 + ilat)
}

// UnpackGrid unpacks a 16-bit value into a grid square using JS8 format
func UnpackGrid(packed uint16) string {
	const nbasegrid = 32400 // Max valid grid value in JS8

	if packed > nbasegrid {
		return ""
	}

	dlat := float32(int(packed)%180 - 90)
	dlong := float32(int(packed)/180*2 - 180 + 2)

	return Deg2Grid(dlong, dlat)
}
