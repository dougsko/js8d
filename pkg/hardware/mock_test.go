package hardware

import (
	"testing"
)

func TestMockGPIO(t *testing.T) {
	gpio := NewMockGPIO()

	t.Run("Initialize", func(t *testing.T) {
		err := gpio.Initialize()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Set and Get Pin", func(t *testing.T) {
		pin := 18

		// Set pin high
		err := gpio.SetPin(pin, true)
		if err != nil {
			t.Errorf("Failed to set pin high: %v", err)
		}

		// Get pin value
		value, err := gpio.GetPin(pin)
		if err != nil {
			t.Errorf("Failed to get pin value: %v", err)
		}
		if !value {
			t.Error("Expected pin to be high")
		}

		// Set pin low
		err = gpio.SetPin(pin, false)
		if err != nil {
			t.Errorf("Failed to set pin low: %v", err)
		}

		// Get pin value
		value, err = gpio.GetPin(pin)
		if err != nil {
			t.Errorf("Failed to get pin value: %v", err)
		}
		if value {
			t.Error("Expected pin to be low")
		}
	})

	t.Run("Multiple Pins", func(t *testing.T) {
		pins := []int{18, 24, 25}
		values := []bool{true, false, true}

		// Set multiple pins
		for i, pin := range pins {
			err := gpio.SetPin(pin, values[i])
			if err != nil {
				t.Errorf("Failed to set pin %d: %v", pin, err)
			}
		}

		// Verify all pins
		for i, pin := range pins {
			value, err := gpio.GetPin(pin)
			if err != nil {
				t.Errorf("Failed to get pin %d: %v", pin, err)
			}
			if value != values[i] {
				t.Errorf("Pin %d: expected %t, got %t", pin, values[i], value)
			}
		}
	})

	t.Run("Unset Pin Default", func(t *testing.T) {
		// Unset pins should default to false
		value, err := gpio.GetPin(99)
		if err != nil {
			t.Errorf("Failed to get unset pin: %v", err)
		}
		if value {
			t.Error("Expected unset pin to be false")
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := gpio.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got: %v", err)
		}
	})
}

func TestMockOLED(t *testing.T) {
	width, height := 128, 64
	oled := NewMockOLED(width, height)

	t.Run("Initialize", func(t *testing.T) {
		err := oled.Initialize()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Dimensions", func(t *testing.T) {
		if oled.GetWidth() != width {
			t.Errorf("Expected width %d, got %d", width, oled.GetWidth())
		}
		if oled.GetHeight() != height {
			t.Errorf("Expected height %d, got %d", height, oled.GetHeight())
		}
	})

	t.Run("Write Lines", func(t *testing.T) {
		lines := []string{
			"K3DEP FN20",
			"14.078 MHz",
			"Last message...",
		}

		for i, text := range lines {
			err := oled.WriteLine(i, text)
			if err != nil {
				t.Errorf("Failed to write line %d: %v", i, err)
			}
		}
	})

	t.Run("Invalid Line Number", func(t *testing.T) {
		// Height 64 means 8 lines (64/8), so line 8 should be invalid
		err := oled.WriteLine(8, "Invalid line")
		if err == nil {
			t.Error("Expected error for invalid line number")
		}

		// Negative line should also be invalid
		err = oled.WriteLine(-1, "Negative line")
		if err == nil {
			t.Error("Expected error for negative line number")
		}
	})

	t.Run("Clear Display", func(t *testing.T) {
		// Write some lines first
		oled.WriteLine(0, "Test line")
		oled.WriteLine(1, "Another line")

		// Clear display
		err := oled.Clear()
		if err != nil {
			t.Errorf("Failed to clear display: %v", err)
		}
	})

	t.Run("Display Update", func(t *testing.T) {
		// Write some content
		oled.WriteLine(0, "Station: K3DEP")
		oled.WriteLine(1, "Freq: 14.078 MHz")

		// Update display
		err := oled.Display()
		if err != nil {
			t.Errorf("Failed to update display: %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := oled.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got: %v", err)
		}
	})
}

func TestMockAudio(t *testing.T) {
	config := MockAudioConfig{
		InputDevice:  "test_input",
		OutputDevice: "test_output",
		SampleRate:   48000,
		BufferSize:   1024,
		Channels:     1,
	}

	audio := NewMockAudio(config)

	t.Run("Initialize", func(t *testing.T) {
		err := audio.Initialize()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Configuration", func(t *testing.T) {
		if audio.GetSampleRate() != config.SampleRate {
			t.Errorf("Expected sample rate %d, got %d", config.SampleRate, audio.GetSampleRate())
		}
		if audio.GetBufferSize() != config.BufferSize {
			t.Errorf("Expected buffer size %d, got %d", config.BufferSize, audio.GetBufferSize())
		}
	})

	t.Run("Default Configuration", func(t *testing.T) {
		// Test with zero values that should get defaults
		defaultConfig := MockAudioConfig{
			InputDevice:  "default_input",
			OutputDevice: "default_output",
		}
		defaultAudio := NewMockAudio(defaultConfig)

		if defaultAudio.GetSampleRate() != 48000 {
			t.Errorf("Expected default sample rate 48000, got %d", defaultAudio.GetSampleRate())
		}
		if defaultAudio.GetBufferSize() != 1024 {
			t.Errorf("Expected default buffer size 1024, got %d", defaultAudio.GetBufferSize())
		}
	})

	t.Run("Initial State", func(t *testing.T) {
		if audio.IsRecording() {
			t.Error("Expected audio to not be recording initially")
		}
		if audio.IsPlaying() {
			t.Error("Expected audio to not be playing initially")
		}
	})

	t.Run("Input Control", func(t *testing.T) {
		// Start input
		err := audio.StartInput()
		if err != nil {
			t.Errorf("Failed to start input: %v", err)
		}
		if !audio.IsRecording() {
			t.Error("Expected audio to be recording after start")
		}

		// Try to start again (should error)
		err = audio.StartInput()
		if err == nil {
			t.Error("Expected error when starting input twice")
		}

		// Stop input
		err = audio.StopInput()
		if err != nil {
			t.Errorf("Failed to stop input: %v", err)
		}
		if audio.IsRecording() {
			t.Error("Expected audio to not be recording after stop")
		}
	})

	t.Run("Output Control", func(t *testing.T) {
		// Start output
		err := audio.StartOutput()
		if err != nil {
			t.Errorf("Failed to start output: %v", err)
		}
		if !audio.IsPlaying() {
			t.Error("Expected audio to be playing after start")
		}

		// Try to start again (should error)
		err = audio.StartOutput()
		if err == nil {
			t.Error("Expected error when starting output twice")
		}

		// Stop output
		err = audio.StopOutput()
		if err != nil {
			t.Errorf("Failed to stop output: %v", err)
		}
		if audio.IsPlaying() {
			t.Error("Expected audio to not be playing after stop")
		}
	})

	t.Run("Play Audio", func(t *testing.T) {
		// Should fail when output not started
		samples := make([]int16, 100)
		err := audio.PlayAudio(samples)
		if err == nil {
			t.Error("Expected error when playing without output started")
		}

		// Start output and try again
		err = audio.StartOutput()
		if err != nil {
			t.Fatalf("Failed to start output: %v", err)
		}

		err = audio.PlayAudio(samples)
		if err != nil {
			t.Errorf("Failed to play audio: %v", err)
		}

		// Test with different sample sizes
		largeSamples := make([]int16, 4096)
		err = audio.PlayAudio(largeSamples)
		if err != nil {
			t.Errorf("Failed to play large audio buffer: %v", err)
		}
	})

	t.Run("Input Samples Channel", func(t *testing.T) {
		samplesChan := audio.GetInputSamples()
		if samplesChan == nil {
			t.Error("Expected non-nil samples channel")
		}

		// Channel should be buffered and not block
		select {
		case <-samplesChan:
			// Should not receive anything initially
			t.Error("Unexpected data in samples channel")
		default:
			// This is expected
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := audio.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got: %v", err)
		}

		// Should stop recording and playing
		if audio.IsRecording() {
			t.Error("Expected recording to stop after close")
		}
		if audio.IsPlaying() {
			t.Error("Expected playing to stop after close")
		}
	})
}

func TestMockAudioConcurrency(t *testing.T) {
	config := MockAudioConfig{
		SampleRate: 48000,
		BufferSize: 1024,
		Channels:   1,
	}

	audio := NewMockAudio(config)
	err := audio.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize audio: %v", err)
	}
	defer audio.Close()

	t.Run("Concurrent State Access", func(t *testing.T) {
		done := make(chan bool, 10)

		// Multiple goroutines reading/writing state
		for i := 0; i < 5; i++ {
			go func() {
				defer func() { done <- true }()
				audio.StartInput()
				audio.IsRecording()
				audio.StopInput()
			}()
		}

		for i := 0; i < 5; i++ {
			go func() {
				defer func() { done <- true }()
				audio.StartOutput()
				audio.IsPlaying()
				audio.StopOutput()
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestMockInterfaces(t *testing.T) {
	t.Run("GPIO Interface Compliance", func(t *testing.T) {
		var _ GPIOInterface = (*MockGPIO)(nil)
	})

	t.Run("OLED Interface Compliance", func(t *testing.T) {
		var _ OLEDInterface = (*MockOLED)(nil)
	})

	t.Run("Audio Interface Compliance", func(t *testing.T) {
		var _ AudioInterface = (*MockAudio)(nil)
	})
}