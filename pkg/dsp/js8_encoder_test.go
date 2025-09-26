package dsp

import (
	"testing"
)

func TestJS8Alphabet(t *testing.T) {
	// Test valid characters
	validChars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-+"

	for i, c := range []byte(validChars) {
		word, err := alphabetWord(c)
		if err != nil {
			t.Errorf("Valid character '%c' failed: %v", c, err)
		}
		if int(word) != i {
			t.Errorf("Character '%c' expected word %d, got %d", c, i, word)
		}
	}

	// Test invalid characters
	invalidChars := " !@#$%^&*()[]{}\\|;:'\",.<>?/`~"
	for _, c := range []byte(invalidChars) {
		_, err := alphabetWord(c)
		if err == nil {
			t.Errorf("Invalid character '%c' should have failed", c)
		}
	}
}

func TestCRC12(t *testing.T) {
	// Test with known data
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	crc := computeCRC12(testData)

	// CRC should be deterministic
	crc2 := computeCRC12(testData)
	if crc != crc2 {
		t.Errorf("CRC not deterministic: %d vs %d", crc, crc2)
	}

	// CRC should be different for different data
	testData2 := []byte{0x01, 0x02, 0x03, 0x05}
	crc3 := computeCRC12(testData2)
	if crc == crc3 {
		t.Errorf("CRC should be different for different data")
	}

	t.Logf("CRC-12 of %v = 0x%03x", testData, crc)
}

func TestMessageValidation(t *testing.T) {
	// Test valid message
	validMsg := "CQ-N0CALL-XX"
	err := ValidateMessage(validMsg)
	if err != nil {
		t.Errorf("Valid message failed validation: %v", err)
	}

	// Test invalid message
	invalidMsg := "CQ N0CALL XX" // contains spaces
	err = ValidateMessage(invalidMsg)
	if err == nil {
		t.Error("Invalid message should have failed validation")
	}
}

func TestMessagePadding(t *testing.T) {
	// Test normal padding
	msg := "CQ"
	padded, err := PadMessage(msg, '-')
	if err != nil {
		t.Errorf("Padding failed: %v", err)
	}
	if len(padded) != 12 {
		t.Errorf("Padded message wrong length: expected 12, got %d", len(padded))
	}
	if padded != "CQ----------" {
		t.Errorf("Unexpected padding result: %s", padded)
	}

	// Test exact length
	exactMsg := "ABCDEFGHIJKL"
	padded2, err := PadMessage(exactMsg, '-')
	if err != nil {
		t.Errorf("Exact length padding failed: %v", err)
	}
	if padded2 != exactMsg {
		t.Errorf("Exact length message should be unchanged")
	}

	// Test too long
	longMsg := "ABCDEFGHIJKLM"
	_, err = PadMessage(longMsg, '-')
	if err == nil {
		t.Error("Too long message should have failed")
	}

	// Test invalid fill character
	_, err = PadMessage("CQ", ' ')
	if err == nil {
		t.Error("Invalid fill character should have failed")
	}
}

func TestJS8Encoder(t *testing.T) {
	encoder := NewJS8Encoder()

	// Test valid encoding
	message := "CQ-N0CALL-XX"
	tones, err := encoder.EncodeMessage(message, 0)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	if len(tones) != 79 {
		t.Errorf("Expected 79 tones, got %d", len(tones))
	}

	// Check Costas arrays are in correct positions
	expectedCostas := []int{4, 2, 5, 6, 1, 3, 0}

	// Check first Costas array (position 0-6)
	for i, expected := range expectedCostas {
		if tones[i] != expected {
			t.Errorf("First Costas[%d]: expected %d, got %d", i, expected, tones[i])
		}
	}

	// Check second Costas array (position 36-42)
	for i, expected := range expectedCostas {
		if tones[36+i] != expected {
			t.Errorf("Second Costas[%d]: expected %d, got %d", i, expected, tones[36+i])
		}
	}

	// Check third Costas array (position 72-78)
	for i, expected := range expectedCostas {
		if tones[72+i] != expected {
			t.Errorf("Third Costas[%d]: expected %d, got %d", i, expected, tones[72+i])
		}
	}

	t.Logf("✓ Encoded message '%s' to %d tones", message, len(tones))
}

func TestAudioGeneration(t *testing.T) {
	encoder := NewJS8Encoder()

	// Test audio generation
	message := "CQ-N0CALL-XX"
	audio, err := encoder.EncodeToAudio(message, 0, 12000)
	if err != nil {
		t.Fatalf("Audio encoding failed: %v", err)
	}

	// Should generate about 15 seconds at 12kHz
	expectedSamples := 15 * 12000
	if len(audio) < expectedSamples-1000 || len(audio) > expectedSamples+1000 {
		t.Errorf("Expected ~%d samples, got %d", expectedSamples, len(audio))
	}

	// Check that audio isn't all zeros
	nonZero := 0
	for _, sample := range audio {
		if sample != 0 {
			nonZero++
		}
	}

	if nonZero == 0 {
		t.Error("Audio should not be all zeros")
	}

	t.Logf("✓ Generated %d audio samples (%d non-zero)", len(audio), nonZero)
}

func TestInvalidMessages(t *testing.T) {
	encoder := NewJS8Encoder()

	// Test wrong length
	_, err := encoder.EncodeMessage("SHORT", 0)
	if err == nil {
		t.Error("Short message should have failed")
	}

	_, err = encoder.EncodeMessage("TOOLONGMESSAGE", 0)
	if err == nil {
		t.Error("Long message should have failed")
	}

	// Test invalid characters
	_, err = encoder.EncodeMessage("CQ N0CALL XX", 0) // spaces
	if err == nil {
		t.Error("Message with spaces should have failed")
	}

	_, err = encoder.EncodeMessage("CQ@N0CALL!XX", 0) // special chars
	if err == nil {
		t.Error("Message with special characters should have failed")
	}
}

func TestEncodingConsistency(t *testing.T) {
	encoder := NewJS8Encoder()
	message := "CQ-N0CALL-XX"

	// Encode the same message multiple times
	tones1, err := encoder.EncodeMessage(message, 0)
	if err != nil {
		t.Fatalf("First encoding failed: %v", err)
	}

	tones2, err := encoder.EncodeMessage(message, 0)
	if err != nil {
		t.Fatalf("Second encoding failed: %v", err)
	}

	// Should be identical
	if len(tones1) != len(tones2) {
		t.Fatalf("Tone arrays different lengths: %d vs %d", len(tones1), len(tones2))
	}

	for i, tone1 := range tones1 {
		if tone1 != tones2[i] {
			t.Errorf("Tone %d differs: %d vs %d", i, tone1, tones2[i])
		}
	}

	t.Log("✓ Encoding is consistent across multiple calls")
}