#include "js8_encoder.h"
#include <string.h>
#include <stdio.h>
#include <stdexcept>
#include <array>
#include <cstdint>

// JS8 alphabet for 6-bit encoding
static const char js8_alphabet[] = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-+";

// Alphabet lookup table (256 bytes, constexpr equivalent)
static uint8_t alphabet_table[256];
static bool alphabet_table_init = false;

// Costas arrays for JS8 Normal mode
static const int costas_normal[3][7] = {
    {4, 2, 5, 6, 1, 3, 0},  // Start
    {4, 2, 5, 6, 1, 3, 0},  // Middle
    {4, 2, 5, 6, 1, 3, 0}   // End
};

// CRC-12 implementation (simplified version of boost::augmented_crc)
// Using polynomial 0xc06 (standard CRC-12 3GPP)
static uint16_t crc12_table[256];
static bool crc12_table_init = false;

// Initialize CRC-12 lookup table
static void init_crc12_table() {
    if (crc12_table_init) return;

    const uint16_t poly = 0xc06;
    for (int i = 0; i < 256; i++) {
        uint16_t crc = i << 4;  // CRC-12, so shift by 4
        for (int j = 0; j < 8; j++) {
            if (crc & 0x800) {
                crc = (crc << 1) ^ poly;
            } else {
                crc <<= 1;
            }
            crc &= 0xfff;  // Keep only 12 bits
        }
        crc12_table[i] = crc;
    }
    crc12_table_init = true;
}

// Compute CRC-12 of byte array
static uint16_t compute_crc12(const uint8_t* data, size_t len) {
    init_crc12_table();

    uint16_t crc = 0;
    for (size_t i = 0; i < len; i++) {
        uint8_t tbl_idx = ((crc >> 4) ^ data[i]) & 0xff;
        crc = ((crc << 8) ^ crc12_table[tbl_idx]) & 0xfff;
    }
    return crc ^ 42;  // XOR with 42 as in original
}

// Initialize alphabet lookup table
static void init_alphabet_table() {
    if (alphabet_table_init) return;

    // Initialize all entries as invalid
    for (int i = 0; i < 256; i++) {
        alphabet_table[i] = 0xff;
    }

    // Set valid alphabet characters
    for (int i = 0; i < 64; i++) {
        alphabet_table[(uint8_t)js8_alphabet[i]] = i;
    }
    alphabet_table_init = true;
}

// Convert character to 6-bit word
static uint8_t alphabet_word(char c) {
    init_alphabet_table();

    uint8_t word = alphabet_table[(uint8_t)c];
    if (word == 0xff) {
        throw std::runtime_error("Invalid character in JS8 message");
    }
    return word;
}

// Simplified parity matrix - this is a critical component that would need
// the full 87x87 matrix from the original code. For now, using a stub.
static bool get_parity_bit(size_t row, size_t col) {
    // TODO: Implement full parity matrix from original JS8.cpp
    // This is a placeholder that will generate incorrect parity but allow testing
    return ((row * 13 + col * 17) % 3) == 0;  // More complex pattern than before
}

// Encode JS8 message to tone sequence
int js8_encode_message(const char* message, int type, int* tones) {
    if (!message || !tones) {
        return -1;
    }

    // Validate message length (must be exactly 12 characters for JS8 Normal)
    size_t msg_len = strlen(message);
    if (msg_len != 12) {
        return -1;
    }

    try {
        // Initialize output
        memset(tones, 0, 79 * sizeof(int));

        // Create 11-byte array for the 87-bit message
        std::array<uint8_t, 11> bytes = {};

        printf("DEBUG: Encoding message: '%.12s'\n", message);

        // Pack 12 characters into 9 bytes (72 bits)
        for (int i = 0, j = 0; i < 12; i += 4, j += 3) {
            printf("DEBUG: Processing chars '%c%c%c%c'\n",
                   message[i], message[i+1], message[i+2], message[i+3]);

            uint32_t words = (alphabet_word(message[i    ]) << 18) |
                           (alphabet_word(message[i + 1]) << 12) |
                           (alphabet_word(message[i + 2]) <<  6) |
                            alphabet_word(message[i + 3]);

            bytes[j    ] = words >> 16;
            bytes[j + 1] = words >>  8;
            bytes[j + 2] = words;
        }

        // Add frame type (3 bits)
        bytes[9] = (type & 0x07) << 5;

        // Compute and add CRC-12
        uint16_t crc = compute_crc12(bytes.data(), bytes.size());
        bytes[9] |= (crc >> 7) & 0x1F;
        bytes[10] = (crc & 0x7F) << 1;

        // Set up tone array pointers
        int* costas_data = tones;
        int* parity_data = tones + 7;
        int* output_data = tones + 43;

        // Add Costas arrays
        for (int i = 0; i < 3; i++) {
            for (int j = 0; j < 7; j++) {
                costas_data[j] = costas_normal[i][j];
            }
            costas_data += 36;  // Move to next Costas position
        }

        // Generate parity and output data (29 3-bit words each)
        size_t output_bits = 0;
        size_t output_byte = 0;
        uint8_t output_mask = 0x80;
        uint8_t output_word = 0;
        uint8_t parity_word = 0;

        for (size_t i = 0; i < 87; i++) {
            // Compute parity bit for position i
            size_t parity_bits = 0;
            size_t parity_byte = 0;
            uint8_t parity_mask = 0x80;

            for (size_t j = 0; j < 87; j++) {
                if (get_parity_bit(i, j) && (bytes[parity_byte] & parity_mask)) {
                    parity_bits++;
                }
                parity_mask = (parity_mask == 1) ? (++parity_byte, 0x80) : (parity_mask >> 1);
            }

            // Accumulate bits
            parity_word = (parity_word << 1) | (parity_bits & 1);
            output_word = (output_word << 1) | ((bytes[output_byte] & output_mask) != 0);
            output_mask = (output_mask == 1) ? (++output_byte, 0x80) : (output_mask >> 1);

            // Output 3-bit words
            if (++output_bits == 3) {
                *parity_data++ = parity_word;
                *output_data++ = output_word;
                parity_word = 0;
                output_word = 0;
                output_bits = 0;
            }
        }

        return 79;  // Total number of tones in JS8 message

    } catch (const std::exception& e) {
        return -1;  // Error in encoding
    }
}