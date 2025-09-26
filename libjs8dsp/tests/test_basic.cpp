#include "js8dsp.h"
#include "varicode.h"
#include <cstdio>
#include <cstring>

int main() {
    printf("JS8DSP Library Test\n");
    printf("Version: %s\n", js8dsp_get_version());

    // Test initialization
    printf("\nTesting library initialization...\n");
    js8dsp_handle_t handle = js8dsp_init(48000, JS8DSP_MODE_NORMAL);
    if (!handle) {
        printf("ERROR: Failed to initialize JS8DSP library\n");
        return 1;
    }
    printf("✓ Library initialized successfully\n");

    // Test varicode encoder
    printf("\nTesting varicode encoder...\n");
    varicode_encoder_t* encoder = varicode_encoder_create();
    if (!encoder) {
        printf("ERROR: Failed to create varicode encoder\n");
        js8dsp_cleanup(handle);
        return 1;
    }
    printf("✓ Varicode encoder created\n");

    // Test message validation
    const char* test_message = "CQ CQ DE N0CALL";
    int valid = varicode_validate_message(encoder, test_message);
    printf("Message '%s' validation: %s\n", test_message, valid ? "VALID" : "INVALID");

    // Test encoding
    char encoded[256];
    int encode_result = varicode_encode_message(encoder, test_message, encoded, sizeof(encoded));
    if (encode_result > 0) {
        printf("✓ Message encoded successfully (%d bytes)\n", encode_result);
        printf("Encoded: %s\n", encoded);

        // Test decoding the same message
        char decoded[256];
        int decode_result = varicode_decode_symbols(encoder, encoded, decoded, sizeof(decoded));
        if (decode_result > 0) {
            printf("✓ Message decoded successfully: '%s'\n", decoded);
            if (strcmp(decoded, test_message) == 0) {
                printf("✓ Round-trip encoding/decoding successful!\n");
            } else {
                printf("⚠ Decoded message differs from original\n");
            }
        } else {
            printf("ERROR: Failed to decode message\n");
        }
    } else {
        printf("ERROR: Failed to encode message\n");
    }

    // Test buffer size calculation
    int buffer_size = js8dsp_get_encode_buffer_size(handle, test_message);
    if (buffer_size > 0) {
        printf("✓ Required buffer size: %d samples\n", buffer_size);
    }

    // Test statistics
    uint32_t decoded, errors;
    js8dsp_result_t stats_result = js8dsp_get_stats(handle, &decoded, &errors);
    if (stats_result == JS8DSP_OK) {
        printf("✓ Statistics: %u decoded, %u errors\n", decoded, errors);
    }

    // Clean up
    varicode_encoder_destroy(encoder);
    js8dsp_cleanup(handle);

    printf("\n✓ All tests completed successfully!\n");
    return 0;
}