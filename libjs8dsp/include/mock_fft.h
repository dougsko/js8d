#ifndef MOCK_FFT_H
#define MOCK_FFT_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>

// Mock FFT types to replace FFTW3 when not available
typedef struct mock_fft_context MockFFTContext;

typedef struct {
    float real;
    float imag;
} mock_complex;

/**
 * Initialize mock FFT context
 * @param size FFT size
 * @return FFT context or NULL on error
 */
MockFFTContext* mock_fft_init(size_t size);

/**
 * Cleanup mock FFT context
 * @param ctx FFT context
 */
void mock_fft_cleanup(MockFFTContext* ctx);

/**
 * Perform forward FFT (mock - just copies input to output)
 * @param ctx FFT context
 * @param input Input samples
 * @param output Output frequency domain
 * @param size Number of samples
 */
void mock_fft_forward(MockFFTContext* ctx, const mock_complex* input,
                     mock_complex* output, size_t size);

/**
 * Perform inverse FFT (mock - just copies input to output)
 * @param ctx FFT context
 * @param input Input frequency domain
 * @param output Output samples
 * @param size Number of samples
 */
void mock_fft_inverse(MockFFTContext* ctx, const mock_complex* input,
                     mock_complex* output, size_t size);

#ifdef __cplusplus
}
#endif

#endif // MOCK_FFT_H