#ifndef JS8_DECODER_H
#define JS8_DECODER_H

#include "js8dsp.h"

#ifdef __cplusplus
extern "C" {
#endif

// Opaque decoder handle
typedef struct js8_decoder js8_decoder_t;

/**
 * Create JS8 decoder instance
 * @param sample_rate Audio sample rate
 * @param mode JS8 mode (0=normal, 1=fast, etc.)
 * @return Decoder handle or NULL on error
 */
js8_decoder_t* js8_decoder_create(int sample_rate, int mode);

/**
 * Destroy decoder instance
 * @param decoder Decoder handle
 */
void js8_decoder_destroy(js8_decoder_t* decoder);

/**
 * Decode audio buffer
 * @param decoder Decoder handle
 * @param audio_buffer Input audio samples
 * @param buffer_size Number of samples
 * @param messages Output messages array
 * @param max_messages Maximum messages to decode
 * @return Number of messages decoded, or negative error code
 */
int js8_decoder_decode(js8_decoder_t* decoder,
                      const float* audio_buffer,
                      size_t buffer_size,
                      js8dsp_decoded_message_t* messages,
                      int max_messages);

#ifdef __cplusplus
}
#endif

#endif // JS8_DECODER_H