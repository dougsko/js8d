#include "js8dsp.h"
#include "js8_encoder.h"
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

// Real JS8 encoding implementation
int js8dsp_encode_message(const char* message,
                         js8dsp_mode_t mode,
                         int16_t* audio_out,
                         int max_samples) {
    if (!message || !audio_out || max_samples <= 0) {
        set_error("Invalid parameters to js8dsp_encode_message");
        return -1;
    }

    clear_error();

    // Validate message length
    size_t msg_len = strlen(message);
    if (msg_len > 12) {
        set_error("Message too long (max 12 characters for JS8 Normal)");
        return -1;
    }

    // Pad message to exactly 12 characters with spaces
    char padded_message[13];
    snprintf(padded_message, sizeof(padded_message), "%-12.12s", message);

    // Generate JS8 tone sequence
    int tones[79];
    int result = js8_encode_message(padded_message, 0, tones);
    if (result < 0) {
        set_error("Failed to encode JS8 message");
        return -1;
    }

    // Convert tones to audio samples
    // JS8 Normal mode: 12000 Hz sample rate, ~15 second duration
    const int sample_rate = 12000;
    const float tone_duration = 15.0f / 79.0f; // ~0.19 seconds per tone
    const int samples_per_tone = (int)(tone_duration * sample_rate);
    const int total_samples = 79 * samples_per_tone;

    if (total_samples > max_samples) {
        set_error("Output buffer too small for JS8 message");
        return -1;
    }

    // Generate FSK audio for each tone
    const float base_freq = 1000.0f; // Base frequency in Hz
    const float freq_spacing = 12000.0f / 2048.0f; // ~5.86 Hz tone spacing
    const float amplitude = 16384.0f; // ~50% of int16_t range

    int sample_idx = 0;
    for (int tone_idx = 0; tone_idx < 79; tone_idx++) {
        float freq = base_freq + (tones[tone_idx] * freq_spacing);

        for (int i = 0; i < samples_per_tone && sample_idx < max_samples; i++) {
            float t = (float)i / sample_rate;
            audio_out[sample_idx++] = (int16_t)(amplitude * sin(2.0f * M_PI * freq * t));
        }
    }

    return sample_idx;
}

// Get last error message
const char* js8dsp_get_error(void) {
    return error_buffer;
}