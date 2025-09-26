/**
 * Advanced Baseline Computation - Extracted from JS8Call
 * Uses Eigen linear algebra for polynomial fitting to noise floor
 *
 * Original: JS8Call project
 * Extraction: js8d project
 */

#include "../include/baseline_computation.h"
#include <algorithm>
#include <numeric>
#include <cmath>

namespace JS8DSP {

void BaselineComputation::computeBaseline(const std::vector<float>& spectrum,
                                        float freq_resolution,
                                        int ia, int ib,
                                        std::vector<float>& baseline) {

    if (spectrum.empty() || ia < 0 || ib >= static_cast<int>(spectrum.size()) || ia >= ib) {
        baseline.assign(spectrum.size(), 0.0f);
        return;
    }

    // Calculate frequency range indices for baseline determination
    auto bmin = static_cast<size_t>(BASELINE_MIN / freq_resolution);
    auto bmax = static_cast<size_t>(BASELINE_MAX / freq_resolution);

    // Clamp to spectrum bounds
    bmin = std::min(bmin, spectrum.size() - 1);
    bmax = std::min(bmax, spectrum.size() - 1);

    if (bmin >= bmax) {
        baseline.assign(spectrum.size(), 0.0f);
        return;
    }

    auto size = bmax - bmin + 1;
    auto arm = size / (2 * BASELINE_NODES.size());

    // Convert power spectrum to dB scale in the baseline region
    std::vector<float> log_spectrum(spectrum.size());
    std::transform(spectrum.begin(), spectrum.end(), log_spectrum.begin(),
                   [](float value) {
                       return 10.0f * std::log10(std::max(value, 1e-10f));
                   });

#ifndef JS8DSP_NO_EIGEN
    // Eigen version - full polynomial fitting
    // Collect lower envelope points using Chebyshev nodes
    for (std::size_t i = 0; i < BASELINE_NODES.size(); ++i) {
        auto node = size * BASELINE_NODES[i];
        auto base = bmin + static_cast<size_t>(std::round(node));

        // Define sampling window around this node
        auto start = static_cast<size_t>(std::max(static_cast<int>(base) - static_cast<int>(arm),
                                                 static_cast<int>(bmin)));
        auto end = std::min(base + arm, bmax);

        if (start >= end || start >= spectrum.size() || end > spectrum.size()) {
            continue;
        }

        // Extract values in this window
        std::vector<float> window_values;
        for (size_t j = start; j < end; ++j) {
            window_values.push_back(log_spectrum[j]);
        }

        if (window_values.empty()) {
            continue;
        }

        // Find the BASELINE_SAMPLE percentile (10th percentile for noise floor)
        auto n = window_values.size() * BASELINE_SAMPLE / 100;
        if (n >= window_values.size()) n = window_values.size() - 1;

        std::nth_element(window_values.begin(),
                        window_values.begin() + n,
                        window_values.end());

        // Store the point (x, y) for polynomial fitting
        p_(i, 0) = node;  // x coordinate (position in baseline range)
        p_(i, 1) = window_values[n];  // y coordinate (dB value)
    }

    // Extract x and y vectors for matrix operations
    Eigen::VectorXd x = p_.col(0);
    Eigen::VectorXd y = p_.col(1);

    // Build Vandermonde matrix for polynomial fitting
    V_.col(0).setOnes();  // x^0 terms
    for (Eigen::Index i = 1; i < V_.cols(); ++i) {
        V_.col(i) = V_.col(i - 1).cwiseProduct(x);  // x^i terms
    }

    // Solve the least squares problem: V * c = y
    // Using QR decomposition for numerical stability
    c_ = V_.colPivHouseholderQr().solve(y);

#else
    // Fallback version without Eigen - simplified baseline computation
    // Collect sample points
    constexpr int num_points = num_nodes;
    for (int i = 0; i < num_points; ++i) {
        auto node = size * BASELINE_NODES[i];
        auto base = bmin + static_cast<size_t>(std::round(node));

        // Define sampling window
        auto start = static_cast<size_t>(std::max(static_cast<int>(base) - static_cast<int>(arm),
                                                 static_cast<int>(bmin)));
        auto end = std::min(base + arm, bmax);

        if (start >= end || start >= spectrum.size() || end > spectrum.size()) {
            p_[i][0] = node;
            p_[i][1] = 0.0;
            continue;
        }

        // Find minimum value in window (simple noise floor estimate)
        float min_val = log_spectrum[start];
        for (size_t j = start; j < end; ++j) {
            min_val = std::min(min_val, log_spectrum[j]);
        }

        p_[i][0] = node;
        p_[i][1] = min_val;
    }

    // Simple polynomial fitting (least squares with normal equations - not as stable as QR)
    // For simplicity, use a linear fit instead of full polynomial
    if (num_points >= 2) {
        double sum_x = 0, sum_y = 0, sum_xx = 0, sum_xy = 0;
        for (int i = 0; i < num_points; ++i) {
            sum_x += p_[i][0];
            sum_y += p_[i][1];
            sum_xx += p_[i][0] * p_[i][0];
            sum_xy += p_[i][0] * p_[i][1];
        }

        double denom = num_points * sum_xx - sum_x * sum_x;
        if (std::abs(denom) > 1e-10) {
            c_[1] = (num_points * sum_xy - sum_x * sum_y) / denom;  // slope
            c_[0] = (sum_y - c_[1] * sum_x) / num_points;          // intercept
        } else {
            c_[0] = sum_y / num_points;  // average
            c_[1] = 0.0;
        }

        // Clear higher order terms
        for (int i = 2; i < num_nodes; ++i) {
            c_[i] = 0.0;
        }
    }
#endif

    // Initialize baseline array
    baseline.assign(spectrum.size(), 0.0f);

    // Function to map index in [ia, ib] to polynomial domain [0, size-1]
    auto mapIndex = [ia, ib, last = static_cast<float>(size - 1)](int i) -> float {
        if (ib == ia) return 0.0f;  // Avoid division by zero
        return (i - ia) * last / static_cast<float>(ib - ia);
    };

    // Evaluate polynomial and fill baseline array
    for (int i = ia; i <= ib && i < static_cast<int>(baseline.size()); ++i) {
        float x_mapped = mapIndex(i);
        baseline[i] = evaluate(x_mapped) + 0.65f;  // Add offset like JS8Call
    }

    // Fill regions outside [ia, ib] with flat extrapolation
    if (ia > 0 && !baseline.empty()) {
        std::fill(baseline.begin(), baseline.begin() + ia, baseline[ia]);
    }
    if (ib < static_cast<int>(baseline.size()) - 1) {
        std::fill(baseline.begin() + ib + 1, baseline.end(), baseline[ib]);
    }
}

void BaselineComputation::computeBaseline(const std::vector<float>& spectrum,
                                        float freq_resolution,
                                        std::vector<float>& baseline) {

    if (spectrum.empty()) {
        baseline.clear();
        return;
    }

    // Use the full spectrum range by default
    int ia = 0;
    int ib = static_cast<int>(spectrum.size()) - 1;

    // But focus the polynomial fitting on the JS8 frequency range
    computeBaseline(spectrum, freq_resolution, ia, ib, baseline);
}

} // namespace JS8DSP