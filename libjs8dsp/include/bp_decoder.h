#ifndef BP_DECODER_H
#define BP_DECODER_H

#include <array>
#include <cstdint>

// BP Decoder constants from JS8Call
namespace BPDSP {
    constexpr int N = 174;              // Total bits
    constexpr int K = 87;               // Message bits
    constexpr int M = N - K;            // Parity check bits (87)
    constexpr int BP_MAX_ITERATIONS = 25;
    constexpr int BP_MAX_CHECKS = 3;    // Max checks per variable node
    constexpr int BP_MAX_ROWS = 7;      // Max rows per check node

    // Parity check matrix structure
    struct CheckNode {
        int valid_neighbors;
        std::array<int, BP_MAX_ROWS> neighbors;
    };

    // Variable node connections
    using VariableChecks = std::array<int, BP_MAX_CHECKS>;

    // The Nm array: which check nodes connect to each variable node
    extern const std::array<VariableChecks, N> Mn;

    // The Mn array: which variable nodes connect to each check node
    extern const std::array<CheckNode, M> Nm;

    // BP Decoder function
    int bpdecode174(const std::array<float, N>& llr,
                   std::array<int8_t, K>& decoded,
                   std::array<int8_t, N>& cw);
}

#endif // BP_DECODER_H