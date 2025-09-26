#ifndef JS8_ENCODER_H
#define JS8_ENCODER_H

#ifdef __cplusplus
extern "C" {
#endif

// Encode a JS8 message to tone sequence
// message: 12-character JS8 message string (must be exactly 12 chars)
// type: JS8 frame type (0-7)
// tones: output array of 79 integers representing tone sequence
// Returns: 79 on success, -1 on error
int js8_encode_message(const char* message, int type, int* tones);

#ifdef __cplusplus
}
#endif

#endif // JS8_ENCODER_H