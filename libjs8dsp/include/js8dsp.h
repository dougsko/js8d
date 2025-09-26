#ifndef JS8DSP_H
#define JS8DSP_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

// JS8DSP API Version
#define JS8DSP_VERSION_MAJOR 1
#define JS8DSP_VERSION_MINOR 0
#define JS8DSP_VERSION_PATCH 0

// Return codes
typedef enum {
    JS8DSP_OK = 0,
    JS8DSP_ERROR = -1,
    JS8DSP_INVALID_PARAM = -2,
    JS8DSP_OUT_OF_MEMORY = -3,
    JS8DSP_NOT_INITIALIZED = -4
} js8dsp_result_t;

// JS8 mode types
typedef enum {
    JS8DSP_MODE_NORMAL = 0,
    JS8DSP_MODE_FAST = 1,
    JS8DSP_MODE_TURBO = 2,
    JS8DSP_MODE_SLOW = 3,
    JS8DSP_MODE_ULTRA = 4
} js8dsp_mode_t;

// Decoded message structure
typedef struct {
    char message[128];          // Decoded message text
    float snr;                  // Signal-to-noise ratio in dB
    float freq_offset;          // Frequency offset in Hz
    uint32_t timestamp;         // Time offset in samples
    int confidence;             // Decoder confidence (0-100)
} js8dsp_decoded_message_t;

// Opaque handle for DSP context
typedef struct js8dsp_context* js8dsp_handle_t;

// Core API functions

/**
 * Initialize the JS8DSP library
 * @param sample_rate Audio sample rate (typically 48000 or 12000)
 * @param mode JS8 mode (normal, fast, turbo, slow, ultra)
 * @return Handle to DSP context, or NULL on error
 */
js8dsp_handle_t js8dsp_init(int sample_rate, js8dsp_mode_t mode);

/**
 * Clean up and free DSP context
 * @param handle DSP context handle
 */
void js8dsp_cleanup(js8dsp_handle_t handle);

/**
 * Decode audio buffer for JS8 messages
 * @param handle DSP context handle
 * @param audio_buffer Input audio samples (float32, mono)
 * @param buffer_size Number of samples in buffer
 * @param messages Output array for decoded messages
 * @param max_messages Maximum number of messages to decode
 * @return Number of messages decoded, or negative error code
 */
int js8dsp_decode_buffer(js8dsp_handle_t handle,
                        const float* audio_buffer,
                        size_t buffer_size,
                        js8dsp_decoded_message_t* messages,
                        int max_messages);

/**
 * Encode message to audio samples
 * @param handle DSP context handle
 * @param message Text message to encode (null-terminated)
 * @param audio_buffer Output buffer for audio samples
 * @param buffer_size Size of output buffer
 * @return Number of samples generated, or negative error code
 */
int js8dsp_encode_message(js8dsp_handle_t handle,
                         const char* message,
                         float* audio_buffer,
                         size_t buffer_size);

/**
 * Get required buffer size for encoding a message
 * @param handle DSP context handle
 * @param message Message to encode
 * @return Required buffer size in samples, or negative error code
 */
int js8dsp_get_encode_buffer_size(js8dsp_handle_t handle, const char* message);

/**
 * Get library version string
 * @return Version string (e.g., "1.0.0")
 */
const char* js8dsp_get_version(void);

/**
 * Get last error message
 * @param handle DSP context handle
 * @return Error message string, or NULL if no error
 */
const char* js8dsp_get_error(js8dsp_handle_t handle);

/**
 * Set decoder sensitivity threshold
 * @param handle DSP context handle
 * @param threshold SNR threshold in dB (lower = more sensitive)
 * @return JS8DSP_OK on success, error code on failure
 */
js8dsp_result_t js8dsp_set_decode_threshold(js8dsp_handle_t handle, float threshold);

/**
 * Get current decoder statistics
 * @param handle DSP context handle
 * @param total_decoded Total messages decoded (output)
 * @param total_errors Total decode errors (output)
 * @return JS8DSP_OK on success, error code on failure
 */
js8dsp_result_t js8dsp_get_stats(js8dsp_handle_t handle,
                                uint32_t* total_decoded,
                                uint32_t* total_errors);

#ifdef __cplusplus
}
#endif

#endif // JS8DSP_H