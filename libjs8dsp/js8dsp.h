#ifndef JS8DSP_H
#define JS8DSP_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stdbool.h>

// JS8 DSP Library - C API for Go CGO integration
// Extracted from JS8Call for js8d daemon

// Maximum message length for JS8
#define JS8DSP_MAX_MESSAGE_LENGTH 1024

// Audio buffer size (typically 15 seconds at 12kHz)
#define JS8DSP_BUFFER_SIZE 180000

// JS8 submodes
typedef enum {
    JS8DSP_MODE_NORMAL = 0,
    JS8DSP_MODE_FAST = 1,
    JS8DSP_MODE_TURBO = 2,
    JS8DSP_MODE_SLOW = 4,
    JS8DSP_MODE_ULTRA = 8
} js8dsp_mode_t;

// Decoded message structure
typedef struct {
    int utc;           // UTC time (use code_time() format)
    int snr;           // Signal-to-noise ratio in dB
    float dt;          // Time offset in seconds
    float frequency;   // Frequency offset in Hz
    char message[JS8DSP_MAX_MESSAGE_LENGTH];  // Decoded message text
    int msg_type;      // Message type
    float quality;     // Decode quality metric
    int mode;          // JS8 submode
} js8dsp_decode_t;

// Callback function for decode results
typedef void (*js8dsp_callback_t)(const js8dsp_decode_t* decode, void* user_data);

// Initialize the DSP library
// Returns: 0 on success, -1 on error
int js8dsp_init(void);

// Cleanup the DSP library
void js8dsp_cleanup(void);

// Decode audio buffer
// audio_data: 16-bit signed audio samples at 12kHz
// samples: number of samples in the buffer
// callback: function to call for each decoded message
// user_data: user data passed to callback
// Returns: number of messages decoded, -1 on error
int js8dsp_decode_buffer(const int16_t* audio_data,
                        int samples,
                        js8dsp_callback_t callback,
                        void* user_data);

// Encode message to audio
// message: text message to encode
// mode: JS8 submode to use
// audio_out: output buffer for 16-bit signed audio samples at 12kHz
// max_samples: maximum samples that can be written to audio_out
// Returns: number of samples generated, -1 on error
int js8dsp_encode_message(const char* message,
                         js8dsp_mode_t mode,
                         int16_t* audio_out,
                         int max_samples);

// Get last error message
const char* js8dsp_get_error(void);

#ifdef __cplusplus
}
#endif

#endif // JS8DSP_H