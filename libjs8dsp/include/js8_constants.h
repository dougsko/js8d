#ifndef JS8_CONSTANTS_H
#define JS8_CONSTANTS_H

#include <cmath>

// JS8 Protocol Constants (extracted from JS8Call)
namespace JS8Constants {

    // Core parameters
    constexpr int         N        = 174;        // Total bits
    constexpr int         K        = 87;         // Message bits
    constexpr int         M        = N - K;      // Check bits (87)
    constexpr int         KK       = 87;         // Information bits (75 + CRC12)
    constexpr int         ND       = 58;         // Data symbols
    constexpr int         NS       = 21;         // Sync symbols (3 @ Costas 7x7)
    constexpr int         NN       = NS + ND;    // Total channel symbols (79)
    constexpr float       ASYNCMIN = 1.5f;       // Minimum sync
    constexpr int         NFSRCH   = 5;          // Search frequency range in Hz (i.e., +/- 2.5 Hz)
    constexpr int         NMAXCAND = 300;        // Maximum number of candidate signals
    constexpr int         NFILT    = 1400;       // Filter length
    constexpr int         NROWS    = 8;
    constexpr int         NFOS     = 2;
    constexpr int         NSSY     = 4;
    constexpr int         NP       = 3200;
    constexpr int         NP2      = 2812;
    constexpr float       TAU      = 2.0f * M_PI;

    // Sample rates and timing
    constexpr int         JS8_RX_SAMPLE_RATE = 12000;   // 12 kHz sample rate
    constexpr int         JS8A_SYMBOL_SAMPLES = 1920;   // Normal mode: 6.4s/79 symbols ≈ 0.08s per symbol
    constexpr int         JS8A_TX_SECONDS = 13;         // Normal mode duration
    constexpr int         JS8B_SYMBOL_SAMPLES = 960;    // Fast mode
    constexpr int         JS8B_TX_SECONDS = 7;          // Fast mode duration
    constexpr int         JS8C_SYMBOL_SAMPLES = 3840;   // Slow mode
    constexpr int         JS8C_TX_SECONDS = 26;         // Slow mode duration

    // JS8 Mode definitions
    enum class Mode {
        NORMAL = 0,     // Mode A: Standard JS8
        FAST = 1,       // Mode B: Fast JS8
        TURBO = 2,      // Mode C: Turbo JS8
        SLOW = 3,       // Mode D: Slow JS8
        ULTRA = 4       // Mode E: Ultra JS8
    };

    // Costas array types
    enum class CostasType {
        ORIGINAL = 0,   // FT8-style Costas arrays
        MODIFIED = 1    // JS8-specific modified arrays
    };

    // Costas arrays (7x7 sync patterns)
    constexpr int COSTAS_ORIGINAL[3][7] = {
        {4, 2, 5, 6, 1, 3, 0},
        {4, 2, 5, 6, 1, 3, 0},
        {4, 2, 5, 6, 1, 3, 0}
    };

    constexpr int COSTAS_MODIFIED[3][7] = {
        {0, 6, 2, 3, 5, 4, 1},
        {1, 5, 0, 2, 3, 6, 4},
        {2, 5, 0, 6, 4, 1, 3}
    };

    // Mode-specific parameters
    struct ModeParams {
        int nsps;        // Samples per symbol
        int ntxdur;      // TX duration in seconds
        int ndownsps;    // Downsampled samples per symbol
        int ndd;         // Filter parameter
        int jz;          // Symbol offset range
        float astart;    // Start delay
        float basesub;   // Baseline subtraction
        CostasType costas; // Which Costas arrays to use
    };

    // Get parameters for specific mode
    constexpr ModeParams getModeParams(Mode mode) {
        switch (mode) {
            case Mode::NORMAL:
                return {JS8A_SYMBOL_SAMPLES, JS8A_TX_SECONDS, 32, 100, 62, 0.5f, 40.0f, CostasType::ORIGINAL};
            case Mode::FAST:
                return {JS8B_SYMBOL_SAMPLES, JS8B_TX_SECONDS, 20, 100, 62, 0.5f, 40.0f, CostasType::MODIFIED};
            case Mode::SLOW:
                return {JS8C_SYMBOL_SAMPLES, JS8C_TX_SECONDS, 50, 100, 62, 0.5f, 40.0f, CostasType::MODIFIED};
            case Mode::TURBO:
                return {480, 4, 16, 100, 62, 0.5f, 40.0f, CostasType::MODIFIED};
            case Mode::ULTRA:
                return {7680, 52, 80, 100, 62, 0.5f, 40.0f, CostasType::MODIFIED};
            default:
                return {JS8A_SYMBOL_SAMPLES, JS8A_TX_SECONDS, 32, 100, 62, 0.5f, 40.0f, CostasType::ORIGINAL};
        }
    }

    // Frequency bins and resolution
    constexpr float getFrequencyResolution(Mode mode) {
        auto params = getModeParams(mode);
        return static_cast<float>(JS8_RX_SAMPLE_RATE) / (params.nsps * NFOS);
    }

    constexpr float getDownsampleRate(Mode mode) {
        auto params = getModeParams(mode);
        return static_cast<float>(JS8_RX_SAMPLE_RATE) / (params.nsps / params.ndownsps);
    }
}

#endif // JS8_CONSTANTS_H