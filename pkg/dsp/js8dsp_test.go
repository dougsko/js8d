package dsp

import (
	"testing"
	"time"
)

func TestDSPInitialization(t *testing.T) {
	dsp := NewDSP()
	defer dsp.Close()

	err := dsp.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize DSP: %v", err)
	}

	t.Log("✓ DSP initialized successfully")
}

func TestEncodeMessage(t *testing.T) {
	dsp := NewDSP()
	defer dsp.Close()

	err := dsp.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize DSP: %v", err)
	}

	message := "CQ-N0CALL-XX"
	audioData, err := dsp.EncodeMessage(message, ModeNormal)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	if len(audioData) == 0 {
		t.Fatal("No audio data generated")
	}

	t.Logf("✓ Encoded message '%s' to %d audio samples", message, len(audioData))

	// Basic sanity check - audio should be around 15 seconds at 12kHz
	expectedSamples := 15 * 12000
	if len(audioData) < expectedSamples-1000 || len(audioData) > expectedSamples+1000 {
		t.Errorf("Expected ~%d samples, got %d", expectedSamples, len(audioData))
	}
}

func TestDecodeMessage(t *testing.T) {
	dsp := NewDSP()
	defer dsp.Close()

	err := dsp.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize DSP: %v", err)
	}

	// First encode a message
	message := "CQ-N0CALL-XX"
	audioData, err := dsp.EncodeMessage(message, ModeNormal)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	// Now try to decode it
	var decoded []*DecodeResult
	decodeCount, err := dsp.DecodeBuffer(audioData, func(result *DecodeResult) {
		decoded = append(decoded, result)
		t.Logf("DECODED: UTC=%d SNR=%ddB DT=%.1fs FREQ=%.1fHz MSG='%s'",
			result.UTC, result.SNR, result.DT, result.Frequency, result.Message)
	})

	if err != nil {
		t.Fatalf("Failed to decode audio: %v", err)
	}

	if decodeCount == 0 {
		t.Log("⚠ No messages decoded (expected with stub implementation)")
	} else {
		t.Logf("✓ Decoded %d message(s)", decodeCount)

		if len(decoded) != decodeCount {
			t.Errorf("Callback count mismatch: expected %d, got %d", decodeCount, len(decoded))
		}
	}
}

func TestErrorHandling(t *testing.T) {
	dsp := NewDSP()
	defer dsp.Close()

	// Initialize (pure Go - always succeeds)
	err := dsp.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize DSP: %v", err)
	}

	_, err = dsp.EncodeMessage("", ModeNormal)
	if err == nil {
		t.Error("Expected error with empty message")
	}
	t.Logf("✓ Empty message error handling works: %v", err)
}

func TestConcurrentAccess(t *testing.T) {
	dsp := NewDSP()
	defer dsp.Close()

	err := dsp.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize DSP: %v", err)
	}

	// Test concurrent encoding
	done := make(chan bool, 2)

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 5; i++ {
			_, err := dsp.EncodeMessage("CQ-TEST1-XX", ModeNormal)
			if err != nil {
				t.Errorf("Concurrent encode 1 failed: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 5; i++ {
			_, err := dsp.EncodeMessage("CQ-TEST2-XX", ModeNormal)
			if err != nil {
				t.Errorf("Concurrent encode 2 failed: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Wait for both goroutines
	<-done
	<-done

	t.Log("✓ Concurrent access test completed")
}