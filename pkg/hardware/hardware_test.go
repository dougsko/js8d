package hardware

import (
	"testing"
	"time"
)

func TestNewHardwareManager(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:     true,
		PTTGPIOPin:     18,
		StatusLEDPin:   24,
		EnableOLED:     true,
		OLEDI2CAddress: 0x3C,
		OLEDWidth:      128,
		OLEDHeight:     64,
		EnableAudio:    true,
		AudioInput:     "default",
		AudioOutput:    "default",
		SampleRate:     48000,
		BufferSize:     1024,
		EnableRadio:    true,
		UseHamlib:      false, // Use mock for testing
		RadioModel:     "1",   // Dummy rig
		RadioDevice:    "",
		RadioBaudRate:  115200,
	}

	manager := NewHardwareManager(config)

	if manager == nil {
		t.Fatal("Expected non-nil hardware manager")
	}

	if manager.config.EnableGPIO != config.EnableGPIO {
		t.Errorf("Expected EnableGPIO %t, got %t", config.EnableGPIO, manager.config.EnableGPIO)
	}

	if manager.initialized {
		t.Error("Expected manager to not be initialized initially")
	}
}

func TestHardwareManagerInitialization(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:     true,
		PTTGPIOPin:     18,
		StatusLEDPin:   24,
		EnableOLED:     true,
		OLEDI2CAddress: 0x3C,
		OLEDWidth:      128,
		OLEDHeight:     64,
		EnableAudio:    false, // Disable audio to avoid platform dependencies
		EnableRadio:    true,
		UseHamlib:      false, // Use mock for testing
		RadioModel:     "1",   // Dummy rig
		RadioDevice:    "",
		RadioBaudRate:  115200,
	}

	manager := NewHardwareManager(config)

	t.Run("Successful Initialization", func(t *testing.T) {
		err := manager.Initialize()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !manager.IsInitialized() {
			t.Error("Expected manager to be initialized")
		}

		// Test cleanup
		defer manager.Close()
	})

	t.Run("Double Initialization", func(t *testing.T) {
		// Should not error on double initialization
		err := manager.Initialize()
		if err != nil {
			t.Errorf("Expected no error on double initialization, got: %v", err)
		}
	})
}

func TestHardwareManagerPTT(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:   true,
		PTTGPIOPin:   18,
		EnableAudio:  false,
		EnableRadio:  false,
		EnableOLED:   false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	t.Run("Initial PTT State", func(t *testing.T) {
		if manager.GetPTT() {
			t.Error("Expected initial PTT state to be false")
		}
	})

	t.Run("Set PTT On", func(t *testing.T) {
		err := manager.SetPTT(true)
		if err != nil {
			t.Errorf("Failed to set PTT on: %v", err)
		}

		if !manager.GetPTT() {
			t.Error("Expected PTT to be on")
		}
	})

	t.Run("Set PTT Off", func(t *testing.T) {
		err := manager.SetPTT(false)
		if err != nil {
			t.Errorf("Failed to set PTT off: %v", err)
		}

		if manager.GetPTT() {
			t.Error("Expected PTT to be off")
		}
	})
}

func TestHardwareManagerStatusLED(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:   true,
		StatusLEDPin: 24,
		EnableAudio:  false,
		EnableRadio:  false,
		EnableOLED:   false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	t.Run("Set Status LED On", func(t *testing.T) {
		err := manager.SetStatusLED(true)
		if err != nil {
			t.Errorf("Failed to set status LED on: %v", err)
		}
	})

	t.Run("Set Status LED Off", func(t *testing.T) {
		err := manager.SetStatusLED(false)
		if err != nil {
			t.Errorf("Failed to set status LED off: %v", err)
		}
	})
}

func TestHardwareManagerOLED(t *testing.T) {
	config := HardwareConfig{
		EnableOLED:     true,
		OLEDI2CAddress: 0x3C,
		OLEDWidth:      128,
		OLEDHeight:     64,
		EnableGPIO:     false,
		EnableAudio:    false,
		EnableRadio:    false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	t.Run("Update OLED Display", func(t *testing.T) {
		err := manager.UpdateOLED("K3DEP", "FN20", 14078000, "Last message test")
		if err != nil {
			t.Errorf("Failed to update OLED: %v", err)
		}
	})

	t.Run("Update OLED with Long Message", func(t *testing.T) {
		longMessage := "This is a very long message that should be truncated for display"
		err := manager.UpdateOLED("K3DEP", "FN20", 14078000, longMessage)
		if err != nil {
			t.Errorf("Failed to update OLED with long message: %v", err)
		}
	})
}

func TestHardwareManagerRadio(t *testing.T) {
	config := HardwareConfig{
		EnableRadio:   true,
		UseHamlib:     false, // Use mock
		RadioModel:    "1",   // Dummy rig
		RadioDevice:   "",
		RadioBaudRate: 115200,
		EnableGPIO:    false,
		EnableAudio:   false,
		EnableOLED:    false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	// Give radio initialization time to complete
	time.Sleep(100 * time.Millisecond)

	if !manager.IsRadioConnected() {
		t.Skip("Radio not connected, skipping radio tests")
	}

	t.Run("Get Radio Frequency", func(t *testing.T) {
		freq, err := manager.GetRadioFrequency()
		if err != nil {
			t.Errorf("Failed to get radio frequency: %v", err)
		}
		if freq <= 0 {
			t.Errorf("Expected positive frequency, got %d", freq)
		}
	})

	t.Run("Set Radio Frequency", func(t *testing.T) {
		testFreq := int64(14078000)
		err := manager.SetRadioFrequency(testFreq)
		if err != nil {
			t.Errorf("Failed to set radio frequency: %v", err)
		}

		// Verify frequency was set
		freq, err := manager.GetRadioFrequency()
		if err != nil {
			t.Errorf("Failed to get radio frequency after set: %v", err)
		}
		if freq != testFreq {
			t.Errorf("Expected frequency %d, got %d", testFreq, freq)
		}
	})

	t.Run("Get Radio Mode", func(t *testing.T) {
		mode, bandwidth, err := manager.GetRadioMode()
		if err != nil {
			t.Errorf("Failed to get radio mode: %v", err)
		}
		if mode == "" {
			t.Error("Expected non-empty mode")
		}
		if bandwidth <= 0 {
			t.Errorf("Expected positive bandwidth, got %d", bandwidth)
		}
	})

	t.Run("Set Radio Mode", func(t *testing.T) {
		err := manager.SetRadioMode("USB", 2400)
		if err != nil {
			t.Errorf("Failed to set radio mode: %v", err)
		}

		// Verify mode was set
		mode, bandwidth, err := manager.GetRadioMode()
		if err != nil {
			t.Errorf("Failed to get radio mode after set: %v", err)
		}
		if mode != "USB" {
			t.Errorf("Expected mode USB, got %s", mode)
		}
		if bandwidth != 2400 {
			t.Errorf("Expected bandwidth 2400, got %d", bandwidth)
		}
	})

	t.Run("Radio PTT Control", func(t *testing.T) {
		// Get initial PTT state
		initialPTT, err := manager.GetRadioPTT()
		if err != nil {
			t.Errorf("Failed to get initial radio PTT: %v", err)
		}

		// Set PTT on
		err = manager.SetRadioPTT(true)
		if err != nil {
			t.Errorf("Failed to set radio PTT on: %v", err)
		}

		ptt, err := manager.GetRadioPTT()
		if err != nil {
			t.Errorf("Failed to get radio PTT after set on: %v", err)
		}
		if !ptt {
			t.Error("Expected PTT to be on")
		}

		// Set PTT off
		err = manager.SetRadioPTT(false)
		if err != nil {
			t.Errorf("Failed to set radio PTT off: %v", err)
		}

		ptt, err = manager.GetRadioPTT()
		if err != nil {
			t.Errorf("Failed to get radio PTT after set off: %v", err)
		}
		if ptt != initialPTT {
			t.Errorf("Expected PTT to return to initial state %t, got %t", initialPTT, ptt)
		}
	})

	t.Run("Get Radio Info", func(t *testing.T) {
		info, err := manager.GetRadioInfo()
		if err != nil {
			t.Errorf("Failed to get radio info: %v", err)
		}
		if info.Model == "" {
			t.Error("Expected non-empty radio model")
		}
	})

	t.Run("Get Radio Power Level", func(t *testing.T) {
		power, err := manager.GetRadioPowerLevel()
		if err != nil {
			t.Errorf("Failed to get radio power level: %v", err)
		}
		if power < 0 || power > 1 {
			t.Errorf("Expected power level between 0 and 1, got %f", power)
		}
	})

	t.Run("Get Radio SWR Level", func(t *testing.T) {
		swr, err := manager.GetRadioSWRLevel()
		if err != nil {
			t.Errorf("Failed to get radio SWR level: %v", err)
		}
		if swr < 1.0 {
			t.Errorf("Expected SWR >= 1.0, got %f", swr)
		}
	})

	t.Run("Get Radio Signal Level", func(t *testing.T) {
		signal, err := manager.GetRadioSignalLevel()
		if err != nil {
			t.Errorf("Failed to get radio signal level: %v", err)
		}
		// Signal level can be negative (dBm)
		if signal < -200 || signal > 100 {
			t.Errorf("Expected reasonable signal level, got %d", signal)
		}
	})
}

func TestHardwareManagerAudio(t *testing.T) {
	config := HardwareConfig{
		EnableAudio:  true,
		AudioInput:   "default",
		AudioOutput:  "default",
		SampleRate:   48000,
		BufferSize:   1024,
		EnableGPIO:   false,
		EnableRadio:  false,
		EnableOLED:   false,
	}

	manager := NewHardwareManager(config)

	// Replace with mock audio for testing
	mockAudioConfig := MockAudioConfig{
		InputDevice:  config.AudioInput,
		OutputDevice: config.AudioOutput,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     1,
	}
	manager.audio = NewMockAudio(mockAudioConfig)
	manager.audio.Initialize()
	manager.initialized = true

	defer manager.Close()

	t.Run("Start Audio Input", func(t *testing.T) {
		err := manager.StartAudioInput()
		if err != nil {
			t.Errorf("Failed to start audio input: %v", err)
		}

		if !manager.audio.IsRecording() {
			t.Error("Expected audio input to be recording")
		}
	})

	t.Run("Stop Audio Input", func(t *testing.T) {
		err := manager.StopAudioInput()
		if err != nil {
			t.Errorf("Failed to stop audio input: %v", err)
		}

		if manager.audio.IsRecording() {
			t.Error("Expected audio input to be stopped")
		}
	})

	t.Run("Start Audio Output", func(t *testing.T) {
		err := manager.StartAudioOutput()
		if err != nil {
			t.Errorf("Failed to start audio output: %v", err)
		}

		if !manager.audio.IsPlaying() {
			t.Error("Expected audio output to be playing")
		}
	})

	t.Run("Stop Audio Output", func(t *testing.T) {
		err := manager.StopAudioOutput()
		if err != nil {
			t.Errorf("Failed to stop audio output: %v", err)
		}

		if manager.audio.IsPlaying() {
			t.Error("Expected audio output to be stopped")
		}
	})

	t.Run("Play Audio", func(t *testing.T) {
		// Start output first
		err := manager.StartAudioOutput()
		if err != nil {
			t.Fatalf("Failed to start audio output: %v", err)
		}

		samples := make([]int16, 1024)
		for i := range samples {
			samples[i] = int16(i % 1000) // Simple test pattern
		}

		err = manager.PlayAudio(samples)
		if err != nil {
			t.Errorf("Failed to play audio: %v", err)
		}
	})

	t.Run("Get Audio Input Samples", func(t *testing.T) {
		samplesChan := manager.GetAudioInputSamples()
		if samplesChan == nil {
			t.Error("Expected non-nil samples channel")
		}
	})

	t.Run("Get Audio Interface", func(t *testing.T) {
		audio := manager.GetAudio()
		if audio == nil {
			t.Error("Expected non-nil audio interface")
		}

		if audio.GetSampleRate() != config.SampleRate {
			t.Errorf("Expected sample rate %d, got %d", config.SampleRate, audio.GetSampleRate())
		}

		if audio.GetBufferSize() != config.BufferSize {
			t.Errorf("Expected buffer size %d, got %d", config.BufferSize, audio.GetBufferSize())
		}
	})
}

func TestHardwareManagerRadioConfig(t *testing.T) {
	config := HardwareConfig{
		EnableRadio:   false, // Start with radio disabled
		EnableGPIO:    false,
		EnableAudio:   false,
		EnableOLED:    false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	t.Run("Update Radio Config", func(t *testing.T) {
		err := manager.UpdateRadioConfig("2028", "/dev/ttyUSB0", 38400, false)
		if err != nil {
			t.Errorf("Failed to update radio config: %v", err)
		}

		config := manager.GetConfig()
		if config.RadioModel != "2028" {
			t.Errorf("Expected radio model 2028, got %s", config.RadioModel)
		}
		if config.RadioDevice != "/dev/ttyUSB0" {
			t.Errorf("Expected radio device /dev/ttyUSB0, got %s", config.RadioDevice)
		}
		if config.RadioBaudRate != 38400 {
			t.Errorf("Expected baud rate 38400, got %d", config.RadioBaudRate)
		}
		if config.UseHamlib != false {
			t.Errorf("Expected UseHamlib false, got %t", config.UseHamlib)
		}
	})

	t.Run("Retry Radio Connection", func(t *testing.T) {
		// UpdateRadioConfig enables radio, so retry should succeed
		err := manager.RetryRadioConnection()
		if err != nil {
			t.Errorf("Expected no error when retrying radio connection after config update, got: %v", err)
		}
	})
}

func TestHardwareManagerDisabledComponents(t *testing.T) {
	// Test manager with all components disabled
	config := HardwareConfig{
		EnableGPIO:  false,
		EnableAudio: false,
		EnableRadio: false,
		EnableOLED:  false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager with disabled components: %v", err)
	}
	defer manager.Close()

	t.Run("PTT with Disabled GPIO", func(t *testing.T) {
		// Should not error, just log
		err := manager.SetPTT(true)
		if err != nil {
			t.Errorf("Expected no error with disabled GPIO, got: %v", err)
		}

		if !manager.GetPTT() {
			t.Error("Expected PTT state to be tracked even with disabled GPIO")
		}
	})

	t.Run("Status LED with Disabled GPIO", func(t *testing.T) {
		// Should not error, just log
		err := manager.SetStatusLED(true)
		if err != nil {
			t.Errorf("Expected no error with disabled GPIO, got: %v", err)
		}
	})

	t.Run("OLED with Disabled OLED", func(t *testing.T) {
		// Should not error, just log
		err := manager.UpdateOLED("K3DEP", "FN20", 14078000, "Test")
		if err != nil {
			t.Errorf("Expected no error with disabled OLED, got: %v", err)
		}
	})

	t.Run("Audio Operations with Disabled Audio", func(t *testing.T) {
		err := manager.StartAudioInput()
		if err == nil {
			t.Error("Expected error when starting audio input with disabled audio")
		}

		err = manager.StartAudioOutput()
		if err == nil {
			t.Error("Expected error when starting audio output with disabled audio")
		}

		samples := make([]int16, 100)
		err = manager.PlayAudio(samples)
		if err == nil {
			t.Error("Expected error when playing audio with disabled audio")
		}

		samplesChan := manager.GetAudioInputSamples()
		if samplesChan != nil {
			t.Error("Expected nil samples channel with disabled audio")
		}
	})

	t.Run("Radio Operations with Disabled Radio", func(t *testing.T) {
		_, err := manager.GetRadioFrequency()
		if err == nil {
			t.Error("Expected error when getting radio frequency with disabled radio")
		}

		err = manager.SetRadioFrequency(14078000)
		if err == nil {
			t.Error("Expected error when setting radio frequency with disabled radio")
		}

		if manager.IsRadioConnected() {
			t.Error("Expected radio not connected with disabled radio")
		}
	})
}

func TestHardwareManagerClose(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:  true,
		PTTGPIOPin:  18,
		EnableOLED:  true,
		OLEDWidth:   128,
		OLEDHeight:  64,
		EnableAudio: false, // Disable to avoid platform dependencies
		EnableRadio: true,
		UseHamlib:   false, // Use mock
		RadioModel:  "1",
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Set PTT on to test cleanup
	manager.SetPTT(true)
	if !manager.GetPTT() {
		t.Error("PTT should be on before close")
	}

	t.Run("Successful Close", func(t *testing.T) {
		err := manager.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got: %v", err)
		}

		if manager.IsInitialized() {
			t.Error("Expected manager to not be initialized after close")
		}

		if manager.GetPTT() {
			t.Error("Expected PTT to be off after close")
		}
	})

	t.Run("Double Close", func(t *testing.T) {
		// Should not error on double close
		err := manager.Close()
		if err != nil {
			t.Errorf("Expected no error on double close, got: %v", err)
		}
	})
}

func TestHardwareManagerConcurrency(t *testing.T) {
	config := HardwareConfig{
		EnableGPIO:  true,
		PTTGPIOPin:  18,
		EnableAudio: false,
		EnableRadio: false,
		EnableOLED:  false,
	}

	manager := NewHardwareManager(config)
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	defer manager.Close()

	t.Run("Concurrent PTT Operations", func(t *testing.T) {
		done := make(chan bool, 10)

		// Multiple goroutines toggling PTT
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer func() { done <- true }()
				state := i%2 == 0
				err := manager.SetPTT(state)
				if err != nil {
					t.Errorf("Goroutine %d failed to set PTT: %v", i, err)
				}
				manager.GetPTT() // Read operation
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("Concurrent Status Reads", func(t *testing.T) {
		done := make(chan bool, 10)

		// Multiple goroutines reading status
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				manager.IsInitialized()
				manager.GetPTT()
				manager.GetConfig()
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}