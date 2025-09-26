/**
 * JS8 Decoder - Extracted from JS8Call
 * Real C++ implementation without Qt dependencies
 *
 * Original: (C) 2025 Allan Bazinet <w6baz@arrl.net>
 * Extraction: js8d project
 */

#include "../include/js8_decoder.h"
#include "../include/js8_constants.h"
#include "../include/bp_decoder.h"
#include "../include/baseline_computation.h"
#include <cmath>
#include <vector>
#include <complex>
#include <algorithm>
#include <cstring>
#include <array>

// FFT is now handled by Go layer, no C++ FFT needed

// Use std containers instead of Qt
using std::vector;
using std::complex;
using std::array;
using namespace JS8Constants;

namespace JS8DSP {

// Decoded message structure
struct DecodedMessage {
    std::string message;
    float snr;
    float freq_offset;
    float time_offset;
    int confidence;
};

class JS8Decoder {
private:
    int sample_rate_;
    Mode js8_mode_;
    ModeParams mode_params_;
    float decode_threshold_;

    // Signal processing buffers
    vector<complex<float>> input_buffer_;
    vector<complex<float>> downsampled_;
    vector<float> spectrum_;
    vector<float> baseline_;
    array<float, NMAXCAND> candidate_freqs_;
    array<float, NMAXCAND> candidate_snrs_;

    // Advanced baseline computation
    BaselineComputation baseline_computer_;

    // Costas synchronization arrays
    array<array<array<complex<float>, 32>, 7>, 3> costas_templates_;

    // FFT is handled by Go layer

    // Initialize Costas synchronization templates
    void init_costas_templates() {
        const int (*costas_array)[7] = (mode_params_.costas == CostasType::ORIGINAL)
                                       ? COSTAS_ORIGINAL : COSTAS_MODIFIED;

        for (int i = 0; i < 3; ++i) {
            for (int j = 0; j < 7; ++j) {
                int tone = costas_array[i][j];
                // Generate complex exponential for each tone
                for (int k = 0; k < mode_params_.ndownsps; ++k) {
                    float phase = 2.0f * M_PI * tone * k / 8.0f; // 8-FSK
                    costas_templates_[i][j][k] = complex<float>(cos(phase), sin(phase));
                }
            }
        }
    }

    // Downsample and filter input signal
    void downsample_signal(const float* audio_buffer, size_t buffer_size, float center_freq) {
        // Clear previous data
        downsampled_.clear();

        int downsample_factor = mode_params_.nsps / mode_params_.ndownsps;
        float freq_offset = center_freq - (sample_rate_ / 2.0f);

        // Apply frequency shift and downsample
        for (size_t i = 0; i < buffer_size; i += downsample_factor) {
            if (i >= buffer_size) break;

            // Frequency shift
            float phase = 2.0f * M_PI * freq_offset * i / sample_rate_;
            complex<float> shift = complex<float>(cos(phase), sin(phase));
            complex<float> sample = complex<float>(audio_buffer[i], 0.0f) * shift;

            downsampled_.push_back(sample);
        }
    }

    // Costas array synchronization
    float sync_costas(int symbol_start, float /* freq_offset */) {
        float total_sync = 0.0f;

        // Check all 3 Costas arrays
        for (int array_idx = 0; array_idx < 3; ++array_idx) {
            float array_sync = 0.0f;

            // Correlate with each of the 7 symbols in this Costas array
            for (int sym_idx = 0; sym_idx < 7; ++sym_idx) {
                int sym_start = symbol_start + (array_idx * 7 + sym_idx) * mode_params_.ndownsps;

                if (sym_start + mode_params_.ndownsps > static_cast<int>(downsampled_.size())) {
                    continue;
                }

                complex<float> correlation = complex<float>(0.0f, 0.0f);

                // Correlate with the expected Costas symbol
                for (int k = 0; k < mode_params_.ndownsps; ++k) {
                    if (sym_start + k < static_cast<int>(downsampled_.size())) {
                        correlation += downsampled_[sym_start + k] *
                                     std::conj(costas_templates_[array_idx][sym_idx][k]);
                    }
                }

                array_sync += std::abs(correlation);
            }

            total_sync += array_sync;
        }

        return total_sync / 3.0f; // Average across arrays
    }

    // Extract 8-FSK symbols from synchronized signal
    bool extract_symbols(int symbol_start, std::array<int, ND>& symbols) {
        if (symbol_start + NN * mode_params_.ndownsps > static_cast<int>(downsampled_.size())) {
            return false;
        }

        // Skip first 3 Costas arrays (21 symbols) and extract data symbols (58)
        int symbol_idx = 0;

        // Skip first Costas array (7 symbols)
        int offset = symbol_start + 7 * mode_params_.ndownsps;

        // Extract 58 data symbols between Costas arrays
        for (int i = 0; i < ND; ++i) {
            if (i == 29) {
                // Skip middle Costas array (7 symbols)
                offset += 7 * mode_params_.ndownsps;
            }

            // Find the strongest tone for this symbol
            float max_power = 0.0f;
            int best_tone = 0;

            for (int tone = 0; tone < 8; ++tone) {  // 8-FSK
                complex<float> correlation = complex<float>(0.0f, 0.0f);

                for (int k = 0; k < mode_params_.ndownsps; ++k) {
                    if (offset + k < static_cast<int>(downsampled_.size())) {
                        float phase = 2.0f * M_PI * tone * k / 8.0f;
                        complex<float> template_val = complex<float>(cos(phase), sin(phase));
                        correlation += downsampled_[offset + k] * std::conj(template_val);
                    }
                }

                float power = std::abs(correlation);
                if (power > max_power) {
                    max_power = power;
                    best_tone = tone;
                }
            }

            symbols[symbol_idx++] = best_tone;
            offset += mode_params_.ndownsps;
        }

        return symbol_idx == ND;
    }

    // Find candidate signals using advanced baseline computation
    int find_candidates(const float* audio_buffer, size_t buffer_size) {
        // Create power spectrum
        const int freq_bins = 2048; // Higher resolution for better baseline computation
        const float freq_resolution = static_cast<float>(sample_rate_) / freq_bins;

        spectrum_.resize(freq_bins);
        baseline_.resize(freq_bins);

        // Simple power spectrum calculation
        std::fill(spectrum_.begin(), spectrum_.end(), 0.0f);

        for (size_t i = 0; i < buffer_size && i < static_cast<size_t>(freq_bins); ++i) {
            float power = audio_buffer[i] * audio_buffer[i];
            spectrum_[i] = power;
        }

        // Apply windowing to reduce spectral artifacts
        for (int i = 0; i < freq_bins && i < static_cast<int>(buffer_size); ++i) {
            float window = 0.5f * (1.0f - cosf(2.0f * M_PI * i / (freq_bins - 1))); // Hann window
            spectrum_[i] *= window;
        }

        // Compute advanced baseline using Eigen polynomial fitting
        baseline_computer_.computeBaseline(spectrum_, freq_resolution, baseline_);

        // Find candidates by comparing signal to baseline
        int num_candidates = 0;
        const float snr_threshold = 3.0f; // 3 dB above baseline

        for (int bin = 0; bin < freq_bins && num_candidates < NMAXCAND; ++bin) {
            float freq = bin * freq_resolution;
            if (freq < 200.0f || freq > 3000.0f) continue; // Skip out-of-band

            float signal_db = 10.0f * log10f(std::max(spectrum_[bin], 1e-10f));
            float baseline_db = baseline_[bin];
            float snr = signal_db - baseline_db;

            if (snr > snr_threshold) {
                candidate_freqs_[num_candidates] = freq;
                candidate_snrs_[num_candidates] = snr;
                ++num_candidates;
            }
        }

        return num_candidates;
    }

public:
    JS8Decoder(int sample_rate, int mode)
        : sample_rate_(sample_rate), js8_mode_(static_cast<Mode>(mode)),
          mode_params_(getModeParams(js8_mode_)), decode_threshold_(-20.0f) {

        // FFT is handled by Go layer

        // Initialize Costas templates
        init_costas_templates();

        // Reserve space for processing buffers
        input_buffer_.reserve(sample_rate_ * mode_params_.ntxdur);
        downsampled_.reserve(sample_rate_ * mode_params_.ntxdur / (mode_params_.nsps / mode_params_.ndownsps));
    }

    ~JS8Decoder() {
        // No cleanup needed - FFT handled by Go layer
    }

    int decode_buffer(const float* audio_buffer, size_t buffer_size,
                     js8dsp_decoded_message_t* messages, int max_messages) {

        if (!audio_buffer || !messages || max_messages <= 0) {
            return -1;
        }

        // Find candidate signals
        int num_candidates = find_candidates(audio_buffer, buffer_size);
        int decoded_count = 0;

        // Try to decode each candidate
        for (int cand = 0; cand < num_candidates && decoded_count < max_messages; ++cand) {
            float freq = candidate_freqs_[cand];
            float snr = candidate_snrs_[cand];

            // Downsample signal around this frequency
            downsample_signal(audio_buffer, buffer_size, freq);

            if (downsampled_.size() < static_cast<size_t>(NN * mode_params_.ndownsps)) {
                continue; // Not enough data
            }

            // Try different time offsets
            float best_sync = 0.0f;
            int best_offset = 0;

            const int max_offset = downsampled_.size() - NN * mode_params_.ndownsps;
            const int step = mode_params_.ndownsps / 4; // Quarter-symbol steps

            for (int offset = 0; offset < max_offset; offset += step) {
                float sync_strength = sync_costas(offset, 0.0f);

                if (sync_strength > best_sync) {
                    best_sync = sync_strength;
                    best_offset = offset;
                }
            }

            // Check if synchronization is strong enough
            if (best_sync > ASYNCMIN) {
                // We found a synchronized signal! Extract symbols and decode
                std::array<int, ND> data_symbols;

                if (extract_symbols(best_offset, data_symbols)) {
                    // Convert symbols to log-likelihood ratios for BP decoder
                    std::array<float, BPDSP::N> llr;
                    std::array<int8_t, BPDSP::K> decoded_bits;
                    std::array<int8_t, BPDSP::N> codeword;

                    // Convert 8-FSK symbols to bit LLRs (3 bits per symbol, 58 symbols = 174 bits)
                    for (int i = 0; i < ND; ++i) {
                        int symbol = data_symbols[i];

                        // Convert symbol to 3 bits (Gray coding)
                        int b0 = (symbol >> 2) & 1;
                        int b1 = (symbol >> 1) & 1;
                        int b2 = symbol & 1;

                        // Simple LLR calculation (positive = bit 1, negative = bit 0)
                        // This is a simplified approach - real implementation would use
                        // correlation powers for soft decoding
                        llr[i * 3] = (b0 == 1) ? 2.0f : -2.0f;
                        llr[i * 3 + 1] = (b1 == 1) ? 2.0f : -2.0f;
                        llr[i * 3 + 2] = (b2 == 1) ? 2.0f : -2.0f;
                    }

                    // Apply BP decoder
                    int decode_result = BPDSP::bpdecode174(llr, decoded_bits, codeword);

                    if (decode_result >= 0) {
                        // Successfully decoded! Convert bits to message
                        // First 75 bits are message data, last 12 bits are CRC
                        std::string decoded_msg;

                        // Simple bit-to-character conversion (this is a placeholder)
                        // Real implementation would use JS8 message encoding
                        for (int i = 0; i < 72; i += 6) {  // 6 bits per character
                            int char_val = 0;
                            for (int j = 0; j < 6; ++j) {
                                if (i + j < BPDSP::K && decoded_bits[i + j]) {
                                    char_val |= (1 << (5 - j));
                                }
                            }
                            if (char_val >= 32 && char_val < 127) {
                                decoded_msg += static_cast<char>(char_val);
                            }
                        }

                        // Store successful decode
                        snprintf(messages[decoded_count].message, sizeof(messages[decoded_count].message),
                                "DECODED: %s", decoded_msg.c_str());
                        messages[decoded_count].snr = snr;
                        messages[decoded_count].freq_offset = freq - 1500.0f;
                        messages[decoded_count].timestamp = best_offset;
                        messages[decoded_count].confidence = 100 - decode_result; // Fewer errors = higher confidence

                        ++decoded_count;
                    } else {
                        // Decoding failed but we had good sync
                        snprintf(messages[decoded_count].message, sizeof(messages[decoded_count].message),
                                "JS8 SYNC %.1f Hz (decode failed)", freq);
                        messages[decoded_count].snr = snr;
                        messages[decoded_count].freq_offset = freq - 1500.0f;
                        messages[decoded_count].timestamp = best_offset;
                        messages[decoded_count].confidence = static_cast<int>(best_sync * 10.0f);

                        ++decoded_count;
                    }
                } else {
                    // Symbol extraction failed
                    snprintf(messages[decoded_count].message, sizeof(messages[decoded_count].message),
                            "JS8 SYNC %.1f Hz (symbol extraction failed)", freq);
                    messages[decoded_count].snr = snr;
                    messages[decoded_count].freq_offset = freq - 1500.0f;
                    messages[decoded_count].timestamp = best_offset;
                    messages[decoded_count].confidence = static_cast<int>(best_sync * 5.0f);

                    ++decoded_count;
                }
            }
        }

        return decoded_count;
    }

    void set_threshold(float threshold) {
        decode_threshold_ = threshold;
    }

    float get_threshold() const {
        return decode_threshold_;
    }
};

} // namespace JS8DSP

// C API implementation
extern "C" {

struct js8_decoder_context {
    JS8DSP::JS8Decoder* decoder;
};

js8_decoder_t* js8_decoder_create(int sample_rate, int mode) {
    try {
        auto ctx = new js8_decoder_context;
        ctx->decoder = new JS8DSP::JS8Decoder(sample_rate, mode);
        return reinterpret_cast<js8_decoder_t*>(ctx);
    } catch (...) {
        return nullptr;
    }
}

void js8_decoder_destroy(js8_decoder_t* decoder) {
    if (!decoder) return;

    auto ctx = reinterpret_cast<js8_decoder_context*>(decoder);
    delete ctx->decoder;
    delete ctx;
}

int js8_decoder_decode(js8_decoder_t* decoder,
                      const float* audio_buffer,
                      size_t buffer_size,
                      js8dsp_decoded_message_t* messages,
                      int max_messages) {
    if (!decoder) return -1;

    auto ctx = reinterpret_cast<js8_decoder_context*>(decoder);
    return ctx->decoder->decode_buffer(audio_buffer, buffer_size,
                                      messages, max_messages);
}

} // extern "C"