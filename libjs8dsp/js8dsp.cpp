#include "js8dsp.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <math.h>

// Static error message buffer
static char error_buffer[256] = {0};

// Set error message
static void set_error(const char* error) {
    strncpy(error_buffer, error, sizeof(error_buffer) - 1);
    error_buffer[sizeof(error_buffer) - 1] = '\0';
}

// Clear error
static void clear_error(void) {
    error_buffer[0] = '\0';
}

// Initialize DSP library
int js8dsp_init(void) {
    clear_error();

    // TODO: Initialize FFTW3 plans, tables, etc.
    // For now, just return success

    return 0;
}

// Cleanup DSP library
void js8dsp_cleanup(void) {
    // TODO: Cleanup FFTW3 plans, free memory, etc.
}

// Simple stub implementation for initial testing
int js8dsp_decode_buffer(const int16_t* audio_data,
                        int samples,
                        js8dsp_callback_t callback,
                        void* user_data) {
    if (!audio_data || samples <= 0 || !callback) {
        set_error("Invalid parameters to js8dsp_decode_buffer");
        return -1;
    }

    clear_error();

    // TODO: Implement actual JS8 decoding
    // For now, create a mock decode result for testing
    js8dsp_decode_t decode = {0};
    decode.utc = 1234;
    decode.snr = 15;
    decode.dt = 0.2f;
    decode.frequency = 1500.0f;
    strncpy(decode.message, "CQ TEST DE N0CALL", sizeof(decode.message) - 1);
    decode.msg_type = 0;
    decode.quality = 0.85f;
    decode.mode = JS8DSP_MODE_NORMAL;

    // Call the callback with mock data
    callback(&decode, user_data);

    return 1; // Return 1 decoded message
}

// Simple stub implementation for encoding
int js8dsp_encode_message(const char* message,
                         js8dsp_mode_t mode,
                         int16_t* audio_out,
                         int max_samples) {
    if (!message || !audio_out || max_samples <= 0) {
        set_error("Invalid parameters to js8dsp_encode_message");
        return -1;
    }

    clear_error();

    // TODO: Implement actual JS8 encoding
    // For now, generate a simple tone burst as placeholder

    int frequency = 1500; // Hz
    int sample_rate = 12000; // Hz
    float duration = 15.0f; // seconds (JS8 Normal mode duration)
    int samples_needed = (int)(duration * sample_rate);

    if (samples_needed > max_samples) {
        set_error("Output buffer too small");
        return -1;
    }

    // Generate a simple sine wave tone
    for (int i = 0; i < samples_needed; i++) {
        float t = (float)i / sample_rate;
        float amplitude = 16384.0f; // ~50% of int16_t range
        audio_out[i] = (int16_t)(amplitude * sin(2.0f * M_PI * frequency * t));
    }

    return samples_needed;
}

// Get last error message
const char* js8dsp_get_error(void) {
    return error_buffer;
}