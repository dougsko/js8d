/**
 * Varicode Encoder/Decoder - Extracted from JS8Call
 * Real C++ implementation without Qt dependencies
 *
 * Original: (C) 2018 Jordan Sherer <kn4crd@gmail.com>
 * Extraction: js8d project
 */

#include "../include/varicode.h"
#include <cstring>
#include <algorithm>
#include <unordered_map>
#include <string>

// JS8 alphabet constants (extracted from original varicode.cpp)
const char* js8_alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ+-./?";
const char* js8_alphabet72 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-+/?.";

namespace JS8DSP {

class VaricodeEncoder {
private:
    std::unordered_map<char, std::string> encode_map_;
    std::unordered_map<std::string, char> decode_map_;

    void initialize_maps() {
        // Actual Huffman varicode table extracted from JS8Call
        const std::pair<char, const char*> huffman_table[] = {
            {' ', "01"},           // Space - most common
            {'E', "100"},          // E - most common letter
            {'T', "1101"},
            {'A', "0011"},
            {'O', "11111"},
            {'I', "11100"},
            {'N', "10111"},
            {'S', "10100"},
            {'H', "00011"},
            {'R', "00000"},
            {'D', "111011"},
            {'L', "110011"},
            {'C', "110001"},
            {'U', "101101"},
            {'M', "101011"},
            {'W', "001011"},
            {'F', "001001"},
            {'G', "000101"},
            {'Y', "000011"},
            {'P', "1111011"},
            {'B', "1111001"},
            {'.', "1110100"},
            {'V', "1100101"},
            {'K', "1100100"},
            {'-', "1100001"},
            {'+', "1100000"},
            {'?', "1011001"},
            {'!', "1011000"},
            {'"', "1010101"},
            {'X', "1010100"},
            {'0', "0010101"},
            {'J', "0010100"},
            {'1', "0010001"},
            {'Q', "0010000"},
            {'2', "0001001"},
            {'Z', "0001000"},
            {'3', "0000101"},
            {'5', "0000100"},
            {'4', "11110101"},
            {'9', "11110100"},
            {'8', "11110001"},
            {'6', "11110000"},
            {'7', "11101011"},
            {'/', "11101010"}
        };

        // Build encode and decode maps
        for (const auto& entry : huffman_table) {
            encode_map_[entry.first] = entry.second;
            decode_map_[entry.second] = entry.first;
        }
    }

public:
    VaricodeEncoder() {
        initialize_maps();
    }

    std::string encode_message(const char* message) {
        if (!message) return "";

        std::string encoded;
        for (const char* p = message; *p; p++) {
            char c = std::toupper(*p);
            auto it = encode_map_.find(c);
            if (it != encode_map_.end()) {
                encoded += it->second;
                // In real varicode, there's no separator - codes are prefix-free
            } else {
                // Handle unknown characters - could skip or use default
                // For now, skip unknown characters
            }
        }
        return encoded;
    }

    std::string decode_symbols(const char* symbols) {
        if (!symbols) return "";

        std::string decoded;
        std::string current_code;

        // Parse bit stream using prefix-free property
        for (const char* p = symbols; *p; p++) {
            if (*p == '0' || *p == '1') {
                current_code += *p;

                // Check if current code matches any character
                auto it = decode_map_.find(current_code);
                if (it != decode_map_.end()) {
                    decoded += it->second;
                    current_code.clear();
                }
                // If code gets too long without match, it might be corrupted
                else if (current_code.length() > 10) {
                    // Skip this bit and try to resync
                    current_code.erase(0, 1);
                }
            }
            // Skip non-binary characters
        }

        return decoded;
    }

    bool is_valid_message(const char* message) {
        if (!message) return false;

        // Basic validation - check if all characters are in alphabet
        for (const char* p = message; *p; p++) {
            char c = std::toupper(*p);
            if (encode_map_.find(c) == encode_map_.end()) {
                return false;
            }
        }
        return true;
    }
};

} // namespace JS8DSP

// C API implementation
extern "C" {

struct varicode_encoder_context {
    JS8DSP::VaricodeEncoder* encoder;
};

varicode_encoder_t* varicode_encoder_create(void) {
    try {
        auto ctx = new varicode_encoder_context;
        ctx->encoder = new JS8DSP::VaricodeEncoder();
        return reinterpret_cast<varicode_encoder_t*>(ctx);
    } catch (...) {
        return nullptr;
    }
}

void varicode_encoder_destroy(varicode_encoder_t* encoder) {
    if (!encoder) return;

    auto ctx = reinterpret_cast<varicode_encoder_context*>(encoder);
    delete ctx->encoder;
    delete ctx;
}

int varicode_encode_message(varicode_encoder_t* encoder,
                           const char* message,
                           char* output,
                           size_t output_size) {
    if (!encoder || !message || !output || output_size == 0) {
        return -1;
    }

    auto ctx = reinterpret_cast<varicode_encoder_context*>(encoder);
    std::string encoded = ctx->encoder->encode_message(message);

    if (encoded.length() >= output_size) {
        return -1; // Buffer too small
    }

    strncpy(output, encoded.c_str(), output_size - 1);
    output[output_size - 1] = '\0';

    return static_cast<int>(encoded.length());
}

int varicode_decode_symbols(varicode_encoder_t* encoder,
                           const char* symbols,
                           char* output,
                           size_t output_size) {
    if (!encoder || !symbols || !output || output_size == 0) {
        return -1;
    }

    auto ctx = reinterpret_cast<varicode_encoder_context*>(encoder);
    std::string decoded = ctx->encoder->decode_symbols(symbols);

    if (decoded.length() >= output_size) {
        return -1; // Buffer too small
    }

    strncpy(output, decoded.c_str(), output_size - 1);
    output[output_size - 1] = '\0';

    return static_cast<int>(decoded.length());
}

int varicode_validate_message(varicode_encoder_t* encoder, const char* message) {
    if (!encoder || !message) return 0;

    auto ctx = reinterpret_cast<varicode_encoder_context*>(encoder);
    return ctx->encoder->is_valid_message(message) ? 1 : 0;
}

} // extern "C"