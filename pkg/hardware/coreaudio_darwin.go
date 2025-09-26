//go:build darwin

package hardware

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AudioToolbox -framework CoreAudio -framework CoreFoundation

#include <AudioToolbox/AudioToolbox.h>
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// Helper structure for audio data
typedef struct {
    int16_t* buffer;
    int capacity;
    int size;
    int readPos;
    int writePos;
} AudioRingBuffer;

// Global variables for audio callback
static AudioRingBuffer inputBuffer = {0};
static AudioRingBuffer outputBuffer = {0};
static AudioUnit inputAudioUnit = NULL;
static AudioUnit outputAudioUnit = NULL;

// Initialize audio buffer
int initAudioBuffer(AudioRingBuffer* buf, int capacity) {
    buf->buffer = malloc(capacity * sizeof(int16_t));
    if (!buf->buffer) return -1;
    buf->capacity = capacity;
    buf->size = 0;
    buf->readPos = 0;
    buf->writePos = 0;
    return 0;
}

// Free audio buffer
void freeAudioBuffer(AudioRingBuffer* buf) {
    if (buf->buffer) {
        free(buf->buffer);
        buf->buffer = NULL;
    }
    buf->capacity = 0;
    buf->size = 0;
    buf->readPos = 0;
    buf->writePos = 0;
}

// Write to audio buffer (thread-safe circular buffer)
int writeAudioBuffer(AudioRingBuffer* buf, int16_t* data, int samples) {
    int available = buf->capacity - buf->size;
    if (samples > available) {
        samples = available; // Truncate if buffer full
    }

    for (int i = 0; i < samples; i++) {
        buf->buffer[buf->writePos] = data[i];
        buf->writePos = (buf->writePos + 1) % buf->capacity;
        buf->size++;
    }

    return samples;
}

// Read from audio buffer (thread-safe circular buffer)
int readAudioBuffer(AudioRingBuffer* buf, int16_t* data, int samples) {
    if (samples > buf->size) {
        samples = buf->size; // Only read what's available
    }

    for (int i = 0; i < samples; i++) {
        data[i] = buf->buffer[buf->readPos];
        buf->readPos = (buf->readPos + 1) % buf->capacity;
        buf->size--;
    }

    return samples;
}

// Input callback for Core Audio
OSStatus inputCallback(void* inRefCon,
                      AudioUnitRenderActionFlags* ioActionFlags,
                      const AudioTimeStamp* inTimeStamp,
                      UInt32 inBusNumber,
                      UInt32 inNumberFrames,
                      AudioBufferList* ioData) {

    // Create buffer list for input
    AudioBufferList bufferList;
    bufferList.mNumberBuffers = 1;
    bufferList.mBuffers[0].mNumberChannels = 1;
    bufferList.mBuffers[0].mDataByteSize = inNumberFrames * sizeof(int16_t);
    bufferList.mBuffers[0].mData = malloc(bufferList.mBuffers[0].mDataByteSize);

    // Render input audio
    OSStatus status = AudioUnitRender(inputAudioUnit, ioActionFlags, inTimeStamp,
                                    inBusNumber, inNumberFrames, &bufferList);

    if (status == noErr) {
        // Convert float samples to int16 and write to buffer
        float* floatSamples = (float*)bufferList.mBuffers[0].mData;
        int16_t* intSamples = malloc(inNumberFrames * sizeof(int16_t));

        for (UInt32 i = 0; i < inNumberFrames; i++) {
            // Convert float (-1.0 to 1.0) to int16 (-32768 to 32767)
            float sample = floatSamples[i];
            if (sample > 1.0f) sample = 1.0f;
            if (sample < -1.0f) sample = -1.0f;
            intSamples[i] = (int16_t)(sample * 32767.0f);
        }

        writeAudioBuffer(&inputBuffer, intSamples, inNumberFrames);
        free(intSamples);
    }

    free(bufferList.mBuffers[0].mData);
    return status;
}

// Output callback for Core Audio
OSStatus outputCallback(void* inRefCon,
                       AudioUnitRenderActionFlags* ioActionFlags,
                       const AudioTimeStamp* inTimeStamp,
                       UInt32 inBusNumber,
                       UInt32 inNumberFrames,
                       AudioBufferList* ioData) {

    // Read samples from output buffer
    int16_t* intSamples = malloc(inNumberFrames * sizeof(int16_t));
    int samplesRead = readAudioBuffer(&outputBuffer, intSamples, inNumberFrames);

    // Convert int16 to float and fill output buffer
    float* floatSamples = (float*)ioData->mBuffers[0].mData;

    for (UInt32 i = 0; i < inNumberFrames; i++) {
        if (i < samplesRead) {
            // Convert int16 to float (-32768 to 32767) -> (-1.0 to 1.0)
            floatSamples[i] = (float)intSamples[i] / 32767.0f;
        } else {
            // Fill with silence if no data available
            floatSamples[i] = 0.0f;
        }
    }

    free(intSamples);
    return noErr;
}

// Initialize Core Audio input
OSStatus initCoreAudioInput(UInt32 sampleRate, UInt32 bufferSize) {
    AudioComponentDescription desc;
    desc.componentType = kAudioUnitType_Output;
    desc.componentSubType = kAudioUnitSubType_HALOutput;
    desc.componentManufacturer = kAudioUnitManufacturer_Apple;
    desc.componentFlags = 0;
    desc.componentFlagsMask = 0;

    AudioComponent component = AudioComponentFindNext(NULL, &desc);
    if (!component) return -1;

    OSStatus status = AudioComponentInstanceNew(component, &inputAudioUnit);
    if (status != noErr) return status;

    // Enable input
    UInt32 enableInput = 1;
    status = AudioUnitSetProperty(inputAudioUnit, kAudioOutputUnitProperty_EnableIO,
                                kAudioUnitScope_Input, 1, &enableInput, sizeof(enableInput));
    if (status != noErr) return status;

    // Disable output
    UInt32 disableOutput = 0;
    status = AudioUnitSetProperty(inputAudioUnit, kAudioOutputUnitProperty_EnableIO,
                                kAudioUnitScope_Output, 0, &disableOutput, sizeof(disableOutput));
    if (status != noErr) return status;

    // Set format
    AudioStreamBasicDescription format;
    format.mSampleRate = sampleRate;
    format.mFormatID = kAudioFormatLinearPCM;
    format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
    format.mBytesPerPacket = sizeof(float);
    format.mFramesPerPacket = 1;
    format.mBytesPerFrame = sizeof(float);
    format.mChannelsPerFrame = 1;
    format.mBitsPerChannel = 32;

    status = AudioUnitSetProperty(inputAudioUnit, kAudioUnitProperty_StreamFormat,
                                kAudioUnitScope_Output, 1, &format, sizeof(format));
    if (status != noErr) return status;

    // Set buffer size
    status = AudioUnitSetProperty(inputAudioUnit, kAudioDevicePropertyBufferFrameSize,
                                kAudioUnitScope_Global, 0, &bufferSize, sizeof(bufferSize));
    if (status != noErr) return status;

    // Set callback
    AURenderCallbackStruct callbackStruct;
    callbackStruct.inputProc = inputCallback;
    callbackStruct.inputProcRefCon = NULL;

    status = AudioUnitSetProperty(inputAudioUnit, kAudioOutputUnitProperty_SetInputCallback,
                                kAudioUnitScope_Global, 0, &callbackStruct, sizeof(callbackStruct));
    if (status != noErr) return status;

    // Initialize buffers
    if (initAudioBuffer(&inputBuffer, sampleRate * 2) != 0) return -1; // 2 second buffer

    return AudioUnitInitialize(inputAudioUnit);
}

// Initialize Core Audio output
OSStatus initCoreAudioOutput(UInt32 sampleRate, UInt32 bufferSize) {
    AudioComponentDescription desc;
    desc.componentType = kAudioUnitType_Output;
    desc.componentSubType = kAudioUnitSubType_DefaultOutput;
    desc.componentManufacturer = kAudioUnitManufacturer_Apple;
    desc.componentFlags = 0;
    desc.componentFlagsMask = 0;

    AudioComponent component = AudioComponentFindNext(NULL, &desc);
    if (!component) return -1;

    OSStatus status = AudioComponentInstanceNew(component, &outputAudioUnit);
    if (status != noErr) return status;

    // Set format
    AudioStreamBasicDescription format;
    format.mSampleRate = sampleRate;
    format.mFormatID = kAudioFormatLinearPCM;
    format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
    format.mBytesPerPacket = sizeof(float);
    format.mFramesPerPacket = 1;
    format.mBytesPerFrame = sizeof(float);
    format.mChannelsPerFrame = 1;
    format.mBitsPerChannel = 32;

    status = AudioUnitSetProperty(outputAudioUnit, kAudioUnitProperty_StreamFormat,
                                kAudioUnitScope_Input, 0, &format, sizeof(format));
    if (status != noErr) return status;

    // Set buffer size
    status = AudioUnitSetProperty(outputAudioUnit, kAudioDevicePropertyBufferFrameSize,
                                kAudioUnitScope_Global, 0, &bufferSize, sizeof(bufferSize));
    if (status != noErr) return status;

    // Set callback
    AURenderCallbackStruct callbackStruct;
    callbackStruct.inputProc = outputCallback;
    callbackStruct.inputProcRefCon = NULL;

    status = AudioUnitSetProperty(outputAudioUnit, kAudioUnitProperty_SetRenderCallback,
                                kAudioUnitScope_Input, 0, &callbackStruct, sizeof(callbackStruct));
    if (status != noErr) return status;

    // Initialize buffers
    if (initAudioBuffer(&outputBuffer, sampleRate * 2) != 0) return -1; // 2 second buffer

    return AudioUnitInitialize(outputAudioUnit);
}

// Start Core Audio input
OSStatus startCoreAudioInput() {
    if (inputAudioUnit == NULL) return -1;
    return AudioOutputUnitStart(inputAudioUnit);
}

// Stop Core Audio input
OSStatus stopCoreAudioInput() {
    if (inputAudioUnit == NULL) return -1;
    return AudioOutputUnitStop(inputAudioUnit);
}

// Start Core Audio output
OSStatus startCoreAudioOutput() {
    if (outputAudioUnit == NULL) return -1;
    return AudioOutputUnitStart(outputAudioUnit);
}

// Stop Core Audio output
OSStatus stopCoreAudioOutput() {
    if (outputAudioUnit == NULL) return -1;
    return AudioOutputUnitStop(outputAudioUnit);
}

// Cleanup Core Audio
void cleanupCoreAudio() {
    if (inputAudioUnit) {
        AudioOutputUnitStop(inputAudioUnit);
        AudioUnitUninitialize(inputAudioUnit);
        AudioComponentInstanceDispose(inputAudioUnit);
        inputAudioUnit = NULL;
    }

    if (outputAudioUnit) {
        AudioOutputUnitStop(outputAudioUnit);
        AudioUnitUninitialize(outputAudioUnit);
        AudioComponentInstanceDispose(outputAudioUnit);
        outputAudioUnit = NULL;
    }

    freeAudioBuffer(&inputBuffer);
    freeAudioBuffer(&outputBuffer);
}

// Read input samples
int readInputSamples(int16_t* buffer, int maxSamples) {
    return readAudioBuffer(&inputBuffer, buffer, maxSamples);
}

// Write output samples
int writeOutputSamples(int16_t* buffer, int samples) {
    return writeAudioBuffer(&outputBuffer, buffer, samples);
}

// Audio device enumeration functions
typedef struct {
    AudioDeviceID deviceID;
    char name[256];
    int isInput;
    int isOutput;
} AudioDeviceInfo;

// Get list of audio devices
int getAudioDevices(AudioDeviceInfo* devices, int maxDevices) {
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
    if (status != noErr) return -1;

    int deviceCount = dataSize / sizeof(AudioDeviceID);
    if (deviceCount > maxDevices) deviceCount = maxDevices;

    AudioDeviceID* deviceIDs = malloc(dataSize);
    status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
    if (status != noErr) {
        free(deviceIDs);
        return -1;
    }

    int validDevices = 0;
    for (int i = 0; i < deviceCount && validDevices < maxDevices; i++) {
        AudioDeviceID deviceID = deviceIDs[i];

        // Get device name
        propertyAddress.mSelector = kAudioDevicePropertyDeviceNameCFString;
        propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
        CFStringRef deviceName = NULL;
        dataSize = sizeof(CFStringRef);

        status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceName);
        if (status != noErr) continue;

        // Convert CFString to C string
        if (!CFStringGetCString(deviceName, devices[validDevices].name, sizeof(devices[validDevices].name), kCFStringEncodingUTF8)) {
            CFRelease(deviceName);
            continue;
        }
        CFRelease(deviceName);

        // Initialize capabilities
        devices[validDevices].isInput = 0;
        devices[validDevices].isOutput = 0;

        // Check if device has input streams
        propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
        propertyAddress.mScope = kAudioDevicePropertyScopeInput;
        dataSize = 0;
        status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
        if (status == noErr && dataSize > 0) {
            AudioBufferList* bufferList = (AudioBufferList*)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            if (status == noErr && bufferList->mNumberBuffers > 0) {
                // Check if any buffer has channels
                for (UInt32 i = 0; i < bufferList->mNumberBuffers; i++) {
                    if (bufferList->mBuffers[i].mNumberChannels > 0) {
                        devices[validDevices].isInput = 1;
                        break;
                    }
                }
            }
            free(bufferList);
        }

        // Check if device has output streams
        propertyAddress.mScope = kAudioDevicePropertyScopeOutput;
        dataSize = 0;
        status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
        if (status == noErr && dataSize > 0) {
            AudioBufferList* bufferList = (AudioBufferList*)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            if (status == noErr && bufferList->mNumberBuffers > 0) {
                // Check if any buffer has channels
                for (UInt32 i = 0; i < bufferList->mNumberBuffers; i++) {
                    if (bufferList->mBuffers[i].mNumberChannels > 0) {
                        devices[validDevices].isOutput = 1;
                        break;
                    }
                }
            }
            free(bufferList);
        }

        devices[validDevices].deviceID = deviceID;
        validDevices++;
    }

    free(deviceIDs);
    return validDevices;
}
*/
import "C"

import (
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

// CoreAudioConfig represents Core Audio configuration
type CoreAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}

// CoreAudio implements real Core Audio I/O for macOS
type CoreAudio struct {
	config CoreAudioConfig

	// State
	recording bool
	playing   bool
	mutex     sync.RWMutex

	// Channels for audio data
	inputSamples  chan []int16
	outputSamples chan []int16

	// Worker control
	stopChan    chan struct{}
	inputWorker chan struct{}
}

// NewCoreAudio creates a new Core Audio interface
func NewCoreAudio(config CoreAudioConfig) *CoreAudio {
	// Set defaults
	if config.SampleRate == 0 {
		config.SampleRate = 48000
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1024
	}
	if config.Channels == 0 {
		config.Channels = 1 // Mono for radio applications
	}

	return &CoreAudio{
		config:        config,
		inputSamples:  make(chan []int16, 10),
		outputSamples: make(chan []int16, 10),
		stopChan:      make(chan struct{}),
		inputWorker:   make(chan struct{}),
	}
}

// Initialize initializes the Core Audio system
func (a *CoreAudio) Initialize() error {
	log.Printf("CoreAudio: Initializing audio system...")
	log.Printf("CoreAudio: Sample rate: %d Hz", a.config.SampleRate)
	log.Printf("CoreAudio: Buffer size: %d samples", a.config.BufferSize)

	// Initialize Core Audio input
	status := C.initCoreAudioInput(C.UInt32(a.config.SampleRate), C.UInt32(a.config.BufferSize))
	if status != 0 {
		return fmt.Errorf("failed to initialize Core Audio input: %d", int(status))
	}

	// Initialize Core Audio output
	status = C.initCoreAudioOutput(C.UInt32(a.config.SampleRate), C.UInt32(a.config.BufferSize))
	if status != 0 {
		C.cleanupCoreAudio()
		return fmt.Errorf("failed to initialize Core Audio output: %d", int(status))
	}

	log.Printf("CoreAudio: Audio system initialized successfully")
	return nil
}

// StartInput starts audio input capture
func (a *CoreAudio) StartInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.recording {
		return fmt.Errorf("audio input already started")
	}

	status := C.startCoreAudioInput()
	if status != 0 {
		return fmt.Errorf("failed to start Core Audio input: %d", int(status))
	}

	a.recording = true
	go a.inputReaderWorker()

	log.Printf("CoreAudio: Audio input started")
	return nil
}

// StopInput stops audio input capture
func (a *CoreAudio) StopInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.recording {
		return nil
	}

	a.recording = false
	close(a.inputWorker)

	status := C.stopCoreAudioInput()
	if status != 0 {
		log.Printf("CoreAudio: Warning - failed to stop input: %d", int(status))
	}

	log.Printf("CoreAudio: Audio input stopped")
	return nil
}

// StartOutput starts audio output
func (a *CoreAudio) StartOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.playing {
		return fmt.Errorf("audio output already started")
	}

	status := C.startCoreAudioOutput()
	if status != 0 {
		return fmt.Errorf("failed to start Core Audio output: %d", int(status))
	}

	a.playing = true
	go a.outputWriterWorker()

	log.Printf("CoreAudio: Audio output started")
	return nil
}

// StopOutput stops audio output
func (a *CoreAudio) StopOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.playing {
		return nil
	}

	a.playing = false

	status := C.stopCoreAudioOutput()
	if status != 0 {
		log.Printf("CoreAudio: Warning - failed to stop output: %d", int(status))
	}

	log.Printf("CoreAudio: Audio output stopped")
	return nil
}

// PlayAudio queues audio samples for output
func (a *CoreAudio) PlayAudio(samples []int16) error {
	if !a.isPlaying() {
		return fmt.Errorf("audio output not started")
	}

	select {
	case a.outputSamples <- samples:
		return nil
	default:
		return fmt.Errorf("audio output buffer full")
	}
}

// GetInputSamples returns a channel for receiving input audio samples
func (a *CoreAudio) GetInputSamples() <-chan []int16 {
	return a.inputSamples
}

// Close shuts down the Core Audio system
func (a *CoreAudio) Close() error {
	// Stop workers
	close(a.stopChan)

	// Stop input/output
	a.StopInput()
	a.StopOutput()

	// Cleanup Core Audio
	C.cleanupCoreAudio()

	// Close channels
	close(a.inputSamples)
	close(a.outputSamples)

	log.Printf("CoreAudio: Audio system closed")
	return nil
}

// inputReaderWorker reads audio from Core Audio input buffer
func (a *CoreAudio) inputReaderWorker() {
	buffer := make([]int16, a.config.BufferSize)

	for a.isRecording() {
		// Read samples from Core Audio buffer
		samplesRead := int(C.readInputSamples((*C.int16_t)(unsafe.Pointer(&buffer[0])), C.int(len(buffer))))

		if samplesRead > 0 {
			// Copy samples to avoid race conditions
			samples := make([]int16, samplesRead)
			copy(samples, buffer[:samplesRead])

			// Send to channel
			select {
			case a.inputSamples <- samples:
			default:
				// Drop samples if buffer full
			}
		}

		// Small delay to prevent busy loop
		time.Sleep(10 * time.Millisecond)

		// Check for stop signal
		select {
		case <-a.inputWorker:
			return
		default:
		}
	}
}

// outputWriterWorker writes audio to Core Audio output buffer
func (a *CoreAudio) outputWriterWorker() {
	for a.isPlaying() {
		select {
		case samples := <-a.outputSamples:
			// Write samples to Core Audio buffer
			samplesWritten := int(C.writeOutputSamples((*C.int16_t)(unsafe.Pointer(&samples[0])), C.int(len(samples))))

			if samplesWritten > 0 {
				log.Printf("CoreAudio: Wrote %d samples to output", samplesWritten)
			}

		case <-a.stopChan:
			return

		case <-time.After(100 * time.Millisecond):
			// Keep the worker alive
			continue
		}
	}
}

// isRecording checks if audio input is active
func (a *CoreAudio) isRecording() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.recording
}

// isPlaying checks if audio output is active
func (a *CoreAudio) isPlaying() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.playing
}

// GetSampleRate returns the current sample rate
func (a *CoreAudio) GetSampleRate() int {
	return a.config.SampleRate
}

// GetBufferSize returns the current buffer size
func (a *CoreAudio) GetBufferSize() int {
	return a.config.BufferSize
}

// IsRecording returns whether audio input is active
func (a *CoreAudio) IsRecording() bool {
	return a.isRecording()
}

// IsPlaying returns whether audio output is active
func (a *CoreAudio) IsPlaying() bool {
	return a.isPlaying()
}

// AudioDevice represents an audio device
type AudioDevice struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	IsInput  bool   `json:"is_input"`
	IsOutput bool   `json:"is_output"`
}

// GetAudioDevices returns a list of available audio devices
func GetAudioDevices() ([]AudioDevice, error) {
	const maxDevices = 64
	devices := make([]C.AudioDeviceInfo, maxDevices)

	log.Printf("CoreAudio: Calling C.getAudioDevices...")
	count := int(C.getAudioDevices(&devices[0], C.int(maxDevices)))
	log.Printf("CoreAudio: C.getAudioDevices returned count=%d", count)

	if count < 0 {
		return nil, fmt.Errorf("failed to enumerate audio devices (returned %d)", count)
	}

	result := make([]AudioDevice, count)
	for i := 0; i < count; i++ {
		name := C.GoString(&devices[i].name[0])
		isInput := devices[i].isInput != 0
		isOutput := devices[i].isOutput != 0

		log.Printf("CoreAudio: Device %d: %s (input:%v, output:%v)", i, name, isInput, isOutput)

		result[i] = AudioDevice{
			ID:       uint32(devices[i].deviceID),
			Name:     name,
			IsInput:  isInput,
			IsOutput: isOutput,
		}
	}

	log.Printf("CoreAudio: Returning %d devices", len(result))
	return result, nil
}