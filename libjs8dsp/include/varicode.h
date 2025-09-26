#ifndef VARICODE_H
#define VARICODE_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// Opaque encoder handle
typedef struct varicode_encoder varicode_encoder_t;

/**
 * Create varicode encoder/decoder instance
 * @return Encoder handle or NULL on error
 */
varicode_encoder_t* varicode_encoder_create(void);

/**
 * Destroy encoder instance
 * @param encoder Encoder handle
 */
void varicode_encoder_destroy(varicode_encoder_t* encoder);

/**
 * Encode text message to varicode symbols
 * @param encoder Encoder handle
 * @param message Input text message
 * @param output Output buffer for encoded symbols
 * @param output_size Size of output buffer
 * @return Length of encoded output, or negative error code
 */
int varicode_encode_message(varicode_encoder_t* encoder,
                           const char* message,
                           char* output,
                           size_t output_size);

/**
 * Decode varicode symbols to text message
 * @param encoder Encoder handle
 * @param symbols Input varicode symbols
 * @param output Output buffer for decoded message
 * @param output_size Size of output buffer
 * @return Length of decoded message, or negative error code
 */
int varicode_decode_symbols(varicode_encoder_t* encoder,
                           const char* symbols,
                           char* output,
                           size_t output_size);

/**
 * Validate that message contains only valid characters
 * @param encoder Encoder handle
 * @param message Message to validate
 * @return 1 if valid, 0 if invalid
 */
int varicode_validate_message(varicode_encoder_t* encoder, const char* message);

#ifdef __cplusplus
}
#endif

#endif // VARICODE_H