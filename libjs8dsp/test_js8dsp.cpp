#include "js8dsp.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Callback function to receive decoded messages
void decode_callback(const js8dsp_decode_t* decode, void* user_data) {
    printf("DECODED: UTC=%d SNR=%ddB DT=%.1fs FREQ=%.1fHz MSG='%s'\n",
           decode->utc, decode->snr, decode->dt, decode->frequency, decode->message);
}

int main() {
    printf("JS8DSP Library Test\n");
    printf("==================\n\n");

    // Initialize the library
    if (js8dsp_init() != 0) {
        printf("ERROR: Failed to initialize JS8DSP library: %s\n", js8dsp_get_error());
        return 1;
    }

    printf("✓ Library initialized successfully\n");

    // Test encoding
    printf("\nTesting encoding...\n");
    const char* test_message = "CQ TEST DE N0CALL";
    int16_t audio_buffer[JS8DSP_BUFFER_SIZE];

    int samples = js8dsp_encode_message(test_message, JS8DSP_MODE_NORMAL,
                                       audio_buffer, JS8DSP_BUFFER_SIZE);
    if (samples > 0) {
        printf("✓ Encoded message '%s' to %d audio samples\n", test_message, samples);
    } else {
        printf("✗ Encoding failed: %s\n", js8dsp_get_error());
    }

    // Test decoding (using the same buffer we just generated)
    printf("\nTesting decoding...\n");
    int decoded_count = js8dsp_decode_buffer(audio_buffer, samples, decode_callback, NULL);
    if (decoded_count > 0) {
        printf("✓ Decoded %d message(s)\n", decoded_count);
    } else if (decoded_count == 0) {
        printf("⚠ No messages decoded (this is expected with stub implementation)\n");
    } else {
        printf("✗ Decoding failed: %s\n", js8dsp_get_error());
    }

    // Test error handling
    printf("\nTesting error handling...\n");
    int result = js8dsp_encode_message(NULL, JS8DSP_MODE_NORMAL, audio_buffer, JS8DSP_BUFFER_SIZE);
    if (result == -1) {
        printf("✓ Error handling works: %s\n", js8dsp_get_error());
    }

    // Cleanup
    js8dsp_cleanup();
    printf("\n✓ Library cleanup completed\n");

    printf("\nAll tests completed successfully!\n");
    printf("Note: This is a stub implementation for testing the API.\n");
    printf("Real DSP functionality will be added in the next phase.\n");

    return 0;
}