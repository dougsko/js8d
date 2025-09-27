#ifndef BASELINE_COMPUTATION_H
#define BASELINE_COMPUTATION_H

#ifndef JS8DSP_NO_EIGEN
#include <Eigen/Dense>
#endif
#include <array>
#include <vector>
#include <cmath>
#include <algorithm>

namespace JS8DSP {

// Baseline computation constants extracted from JS8Call
constexpr int BASELINE_DEGREE = 5;
constexpr int BASELINE_SAMPLE = 10;  // Percentile for sampling
constexpr float BASELINE_MIN = 500.0f;   // Hz
constexpr float BASELINE_MAX = 2500.0f;  // Hz

// Constexpr cos function for Chebyshev nodes (from JS8Call)
constexpr auto cos_approx = [](double const x) {
    constexpr std::array coefficients = {
        1.0,                             // Coefficient for x^0
       -0.49999999999999994,             // Coefficient for x^2
        0.041666666666666664,            // Coefficient for x^4
       -0.001388888888888889,            // Coefficient for x^6
        0.000024801587301587,            // Coefficient for x^8
       -0.00000027557319223986,          // Coefficient for x^10
        0.00000000208767569878681,       // Coefficient for x^12
       -0.00000000001147074513875176,    // Coefficient for x^14
        0.0000000000000477947733238733   // Coefficient for x^16
    };

    auto const x2  = x   * x;
    auto const x4  = x2  * x2;
    auto const x6  = x4  * x2;
    auto const x8  = x4  * x4;
    auto const x10 = x8  * x2;
    auto const x12 = x8  * x4;
    auto const x14 = x12 * x2;
    auto const x16 = x8  * x8;

    return coefficients[0]
         + coefficients[1] * x2
         + coefficients[2] * x4
         + coefficients[3] * x6
         + coefficients[4] * x8
         + coefficients[5] * x10
         + coefficients[6] * x12
         + coefficients[7] * x14
         + coefficients[8] * x16;
};

// Chebyshev nodes generation (from JS8Call)
constexpr auto BASELINE_NODES = []() {
    auto nodes = std::array<double, BASELINE_DEGREE + 1>{};
    constexpr auto slice = M_PI / (2.0 * nodes.size());

    for (std::size_t i = 0; i < nodes.size(); ++i) {
        nodes[i] = 0.5 * (1.0 - cos_approx(slice * (2.0 * i + 1)));
    }
    return nodes;
}();

/**
 * Advanced baseline computation using Eigen linear algebra (when available)
 * Extracted from JS8Call's baselinejs8() function
 */
class BaselineComputation {
public:
#ifndef JS8DSP_NO_EIGEN
    // Type definitions from JS8Call - Eigen version
    using Points = Eigen::Matrix<double, BASELINE_NODES.size(), 2>;
    using Vandermonde = Eigen::Matrix<double, BASELINE_NODES.size(), BASELINE_NODES.size()>;
    using Coefficients = Eigen::Vector<double, BASELINE_NODES.size()>;

private:
    Points p_;
    Vandermonde V_;
    Coefficients c_;

    // Polynomial evaluation using Estrin's method (from JS8Call)
    inline float evaluate(float x) const {
        auto baseline = 0.0;
        auto exponent = 1.0;

        // Unrolled polynomial evaluation
        for (size_t i = 0; i < BASELINE_NODES.size() / 2; ++i) {
            baseline += (c_[i * 2] + c_[i * 2 + 1] * x) * exponent;
            exponent *= x * x;
        }

        return static_cast<float>(baseline);
    }
#else
    // Fallback version without Eigen - uses simple arrays
    static constexpr int num_nodes = BASELINE_DEGREE + 1;

private:
    std::array<std::array<double, 2>, num_nodes> p_;
    std::array<double, num_nodes> c_;

    // Simple polynomial evaluation
    inline float evaluate(float x) const {
        double result = 0.0;
        double x_power = 1.0;

        for (int i = 0; i < num_nodes; ++i) {
            result += c_[i] * x_power;
            x_power *= x;
        }

        return static_cast<float>(result);
    }
#endif

public:
    /**
     * Compute baseline using polynomial fitting to power spectrum
     * @param spectrum Power spectrum data
     * @param freq_resolution Frequency resolution (Hz per bin)
     * @param ia Start index for baseline region
     * @param ib End index for baseline region
     * @param baseline Output baseline array (same size as spectrum)
     */
    void computeBaseline(const std::vector<float>& spectrum,
                        float freq_resolution,
                        int ia, int ib,
                        std::vector<float>& baseline);

    /**
     * Simplified version that processes the standard JS8 frequency range
     * @param spectrum Input power spectrum
     * @param freq_resolution Frequency resolution (Hz per bin)
     * @param baseline Output baseline
     */
    void computeBaseline(const std::vector<float>& spectrum,
                        float freq_resolution,
                        std::vector<float>& baseline);
};

} // namespace JS8DSP

#endif // BASELINE_COMPUTATION_H