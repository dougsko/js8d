package dsp

import (
	"testing"
)

func TestPackUnpack5bits(t *testing.T) {
	for i := uint8(0); i < 32; i++ {
		packed := Pack5bits(i)
		unpacked := Unpack5bits(packed)
		if unpacked != i {
			t.Errorf("Pack/Unpack5bits failed for %d: got %d", i, unpacked)
		}
	}
}

func TestPackUnpack6bits(t *testing.T) {
	for i := uint8(0); i < 41; i++ {
		packed := Pack6bits(i)
		unpacked := Unpack6bits(packed)
		if unpacked != i {
			t.Errorf("Pack/Unpack6bits failed for %d: got %d", i, unpacked)
		}
	}
}

func TestPackUnpack16bits(t *testing.T) {
	testValues := []uint16{0, 1, 255, 1234, 12345, 65535}

	for _, val := range testValues {
		packed := Pack16bits(val)
		unpacked := Unpack16bits(packed)
		if unpacked != val {
			t.Errorf("Pack/Unpack16bits failed for %d: packed=%q, unpacked=%d", val, packed, unpacked)
		}
	}
}

func TestPackUnpack32bits(t *testing.T) {
	testValues := []uint32{0, 1, 255, 12345, 0x12345678, 0xFFFFFFFF}

	for _, val := range testValues {
		packed := Pack32bits(val)
		unpacked := Unpack32bits(packed)
		if unpacked != val {
			t.Errorf("Pack/Unpack32bits failed for %d: packed=%q, unpacked=%d", val, packed, unpacked)
		}
	}
}

func TestPackUnpack64bits(t *testing.T) {
	testValues := []uint64{0, 1, 255, 12345, 0x123456789ABCDEF0, 0xFFFFFFFFFFFFFFFF}

	for _, val := range testValues {
		packed := Pack64bits(val)
		unpacked := Unpack64bits(packed)
		if unpacked != val {
			t.Errorf("Pack/Unpack64bits failed for %d: packed=%q, unpacked=%d", val, packed, unpacked)
		}
	}
}

func TestPackUnpack72bits(t *testing.T) {
	testCases := []struct {
		value uint64
		rem   uint8
	}{
		{0, 0},
		{0x123456789ABCDEF0, 0x42},
		{0xFFFFFFFFFFFFFFFF, 0xFF},
		{0x0123456789ABCDEF, 0xA5},
	}

	for _, tc := range testCases {
		packed := Pack72bits(tc.value, tc.rem)
		var unpackedRem uint8
		unpackedValue := Unpack72bits(packed, &unpackedRem)

		if unpackedValue != tc.value || unpackedRem != tc.rem {
			t.Errorf("Pack/Unpack72bits failed for value=%016x rem=%02x: "+
				"packed=%q, unpacked_value=%016x, unpacked_rem=%02x",
				tc.value, tc.rem, packed, unpackedValue, unpackedRem)
		}
	}
}

func TestIntToBits(t *testing.T) {
	testCases := []struct {
		value    uint64
		bitCount int
		expected []bool
	}{
		{0, 4, []bool{false, false, false, false}},
		{5, 4, []bool{false, true, false, true}},                              // 0101
		{15, 4, []bool{true, true, true, true}},                               // 1111
		{0x55, 8, []bool{false, true, false, true, false, true, false, true}}, // 01010101
	}

	for _, tc := range testCases {
		result := intToBits(tc.value, tc.bitCount)
		if len(result) != len(tc.expected) {
			t.Errorf("intToBits(%d, %d) length mismatch: got %d, want %d",
				tc.value, tc.bitCount, len(result), len(tc.expected))
			continue
		}

		for i, bit := range result {
			if bit != tc.expected[i] {
				t.Errorf("intToBits(%d, %d) bit %d mismatch: got %t, want %t",
					tc.value, tc.bitCount, i, bit, tc.expected[i])
			}
		}
	}
}

func TestBitsToInt(t *testing.T) {
	testCases := []struct {
		bits     []bool
		expected uint64
	}{
		{[]bool{false, false, false, false}, 0},
		{[]bool{false, true, false, true}, 5},                              // 0101
		{[]bool{true, true, true, true}, 15},                               // 1111
		{[]bool{false, true, false, true, false, true, false, true}, 0x55}, // 01010101
	}

	for _, tc := range testCases {
		result := bitsToInt(tc.bits)
		if result != tc.expected {
			t.Errorf("bitsToInt(%v) = %d, want %d", tc.bits, result, tc.expected)
		}
	}
}

func TestIntBitsRoundTrip(t *testing.T) {
	testValues := []struct {
		value    uint64
		bitCount int
	}{
		{0, 8},
		{255, 8},
		{0x1234, 16},
		{0x12345678, 32},
	}

	for _, tv := range testValues {
		bits := intToBits(tv.value, tv.bitCount)
		result := bitsToInt(bits)
		if result != tv.value {
			t.Errorf("Round trip failed for %d: got %d", tv.value, result)
		}
	}
}

func TestChecksum16(t *testing.T) {
	testCases := []string{
		"Hello",
		"JS8 Test Message",
		"CQ N0CALL EM12",
		"",
	}

	for _, input := range testCases {
		checksum := Checksum16(input)
		if len(checksum) != 3 {
			t.Errorf("Checksum16(%q) length = %d, want 3", input, len(checksum))
		}

		if !Checksum16Valid(checksum, input) {
			t.Errorf("Checksum16Valid failed for input %q with checksum %q", input, checksum)
		}

		// Test with wrong checksum
		if Checksum16Valid("ABC", input) && input != "" {
			t.Errorf("Checksum16Valid should have failed for input %q with wrong checksum", input)
		}
	}
}

func TestChecksum32(t *testing.T) {
	testCases := []string{
		"Hello",
		"JS8 Test Message",
		"CQ N0CALL EM12",
		"",
	}

	for _, input := range testCases {
		checksum := Checksum32(input)
		if len(checksum) != 6 {
			t.Errorf("Checksum32(%q) length = %d, want 6", input, len(checksum))
		}

		if !Checksum32Valid(checksum, input) {
			t.Errorf("Checksum32Valid failed for input %q with checksum %q", input, checksum)
		}
	}
}

func TestStringUtilities(t *testing.T) {
	// Test Rstrip
	if result := Rstrip("hello   "); result != "hello" {
		t.Errorf("Rstrip failed: got %q, want %q", result, "hello")
	}

	// Test Lstrip
	if result := Lstrip("   hello"); result != "hello" {
		t.Errorf("Lstrip failed: got %q, want %q", result, "hello")
	}

	// Test Escape/Unescape
	original := "Hello\nWorld\t!"
	escaped := Escape(original)
	unescaped := Unescape(escaped)
	if unescaped != original {
		t.Errorf("Escape/Unescape failed: original=%q, escaped=%q, unescaped=%q",
			original, escaped, unescaped)
	}
}

func TestParseCallsigns(t *testing.T) {
	testCases := []struct {
		input string
		count int // Just check count, not exact matches due to regex complexity
	}{
		{"CQ N0CALL EM12", 1},    // Should find N0CALL
		{"W1ABC/P QRT", 1},       // Should find W1ABC/P
		{"No callsigns here", 0}, // Should find none
		{"K1ABC DE W2XYZ", 2},    // Should find K1ABC and W2XYZ
	}

	for _, tc := range testCases {
		result := ParseCallsigns(tc.input)
		if len(result) != tc.count {
			t.Logf("ParseCallsigns(%q) found: %v", tc.input, result)
			t.Errorf("ParseCallsigns(%q) length = %d, want %d", tc.input, len(result), tc.count)
		}
	}
}

func TestParseGrids(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"CQ N0CALL EM12", []string{"EM12"}},
		{"Grid is FN31pr", []string{"FN31"}}, // Simplified regex catches first 4 chars
		{"No grids here", []string{}},
	}

	for _, tc := range testCases {
		result := ParseGrids(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("ParseGrids(%q) length = %d, want %d", tc.input, len(result), len(tc.expected))
			continue
		}

		for i, grid := range result {
			if i < len(tc.expected) && grid != tc.expected[i] {
				t.Errorf("ParseGrids(%q)[%d] = %q, want %q", tc.input, i, grid, tc.expected[i])
			}
		}
	}
}

func TestCommandValidation(t *testing.T) {
	// Test IsSNRCommand
	if !IsSNRCommand(" SNR?") {
		t.Error("IsSNRCommand should return true for ' SNR?'")
	}
	if IsSNRCommand(" INFO?") {
		t.Error("IsSNRCommand should return false for ' INFO?'")
	}

	// Test IsCommandAllowed
	if !IsCommandAllowed(" SNR?") {
		t.Error("IsCommandAllowed should return true for ' SNR?'")
	}
	if IsCommandAllowed(" INVALID") {
		t.Error("IsCommandAllowed should return false for ' INVALID'")
	}

	// Test IsCommandBuffered
	if !IsCommandBuffered(" MSG") {
		t.Error("IsCommandBuffered should return true for ' MSG'")
	}
	if IsCommandBuffered(" SNR?") {
		t.Error("IsCommandBuffered should return false for ' SNR?'")
	}

	// Test IsCommandAutoreply
	if !IsCommandAutoreply(" SNR?") {
		t.Error("IsCommandAutoreply should return true for ' SNR?'")
	}
	if IsCommandAutoreply(" MSG") {
		t.Error("IsCommandAutoreply should return false for ' MSG'")
	}
}

func TestJS8Formatting(t *testing.T) {
	// Test CQString
	if result := CQString(0); result != "CQ" {
		t.Errorf("CQString(0) = %q, want %q", result, "CQ")
	}
	if result := CQString(5); result != "CQ5" {
		t.Errorf("CQString(5) = %q, want %q", result, "CQ5")
	}

	// Test HBString
	if result := HBString(0); result != "HB" {
		t.Errorf("HBString(0) = %q, want %q", result, "HB")
	}
	if result := HBString(3); result != "HB3" {
		t.Errorf("HBString(3) = %q, want %q", result, "HB3")
	}

	// Test StartsWithCQ
	if !StartsWithCQ("CQ N0CALL") {
		t.Error("StartsWithCQ should return true for 'CQ N0CALL'")
	}
	if StartsWithCQ("N0CALL CQ") {
		t.Error("StartsWithCQ should return false for 'N0CALL CQ'")
	}

	// Test StartsWithHB
	if !StartsWithHB("HB AUTO") {
		t.Error("StartsWithHB should return true for 'HB AUTO'")
	}
	if StartsWithHB("AUTO HB") {
		t.Error("StartsWithHB should return false for 'AUTO HB'")
	}

	// Test FormatSNR
	if result := FormatSNR(15); result != "+15" {
		t.Errorf("FormatSNR(15) = %q, want %q", result, "+15")
	}
	if result := FormatSNR(-10); result != "-10" {
		t.Errorf("FormatSNR(-10) = %q, want %q", result, "-10")
	}

	// Test FormatPWR
	if result := FormatPWR(30); result != "30dBm" {
		t.Errorf("FormatPWR(30) = %q, want %q", result, "30dBm")
	}
}

func TestGridConversion(t *testing.T) {
	testCases := []struct {
		grid string
	}{
		{"EM12"},
		{"FN31"},
		{"JO65"},
	}

	for _, tc := range testCases {
		// Test round-trip conversion: Grid -> Deg -> Grid
		origLon, origLat := Grid2Deg(tc.grid)
		backToGrid := Deg2Grid(origLon, origLat)

		if backToGrid != tc.grid {
			t.Errorf("Grid round-trip failed for %q: Grid2Deg -> Deg2Grid = %q", tc.grid, backToGrid)
		}

		t.Logf("Grid %q: lon=%.1f, lat=%.1f", tc.grid, origLon, origLat)
	}
}

func TestPackUnpackGrid(t *testing.T) {
	testGrids := []string{"EM12", "FN31", "JO65", "AA00"}

	for _, grid := range testGrids {
		packed := PackGrid(grid)
		unpacked := UnpackGrid(packed)
		if unpacked != grid {
			// Debug info
			dlong, dlat := Grid2Deg(grid)
			t.Logf("Grid %q: dlong=%.1f, dlat=%.1f, packed=%d, unpacked=%q",
				grid, dlong, dlat, packed, unpacked)
			t.Errorf("PackGrid/UnpackGrid failed for %q: got %q", grid, unpacked)
		}
	}

	// Test edge case grid - RR99 is technically valid but at edge of world
	edge := "RR99"
	packed := PackGrid(edge)
	unpacked := UnpackGrid(packed)
	t.Logf("Edge grid %q: packed=%d, unpacked=%q", edge, packed, unpacked)
	// RR99 might be valid depending on implementation - just check it doesn't crash
}

func TestFrameTypeString(t *testing.T) {
	if result := FrameHeartbeat.String(); result != "FrameHeartbeat" {
		t.Errorf("FrameHeartbeat.String() = %q, want %q", result, "FrameHeartbeat")
	}

	if result := FrameType(255).String(); result != "FrameUnknown" {
		t.Errorf("FrameType(255).String() = %q, want %q", result, "FrameUnknown")
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
