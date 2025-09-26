#include "../include/mock_fft.h"
#include <cstring>
#include <cstdlib>

struct mock_fft_context {
    size_t size;
    mock_complex* work_buffer;
};

MockFFTContext* mock_fft_init(size_t size) {
    MockFFTContext* ctx = (MockFFTContext*)malloc(sizeof(MockFFTContext));
    if (!ctx) return nullptr;

    ctx->size = size;
    ctx->work_buffer = (mock_complex*)malloc(size * sizeof(mock_complex));
    if (!ctx->work_buffer) {
        free(ctx);
        return nullptr;
    }

    return ctx;
}

void mock_fft_cleanup(MockFFTContext* ctx) {
    if (!ctx) return;

    if (ctx->work_buffer) {
        free(ctx->work_buffer);
    }
    free(ctx);
}

void mock_fft_forward(MockFFTContext* ctx, const mock_complex* input,
                     mock_complex* output, size_t size) {
    if (!ctx || !input || !output) return;

    // Mock implementation - just copy input to output
    // In a real implementation, this would perform the actual FFT
    memcpy(output, input, size * sizeof(mock_complex));
}

void mock_fft_inverse(MockFFTContext* ctx, const mock_complex* input,
                     mock_complex* output, size_t size) {
    if (!ctx || !input || !output) return;

    // Mock implementation - just copy input to output
    // In a real implementation, this would perform the actual IFFT
    memcpy(output, input, size * sizeof(mock_complex));
}