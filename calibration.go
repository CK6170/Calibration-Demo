package main

import (
	"errors"
	"fmt"
)

// ComputeFactors performs a least-squares fit to compute 4 scale factors f0..f3
// such that for each calibration measurement row i:
//
//	yi ≈ sum_j f_j * (adc_ij - zero_j)
//
// Where yi is the known calibration weight (cal.CalibrationWeight) for each placement.
// We construct X (m x 4) where each row is delta ADC, y is a length-m vector (all W).
// We solve (X^T X) f = X^T y for f using Gaussian elimination on the 4x4 normal matrix.
func ComputeFactors(cal CalibrationData) ([4]float64, error) {
	var factors [4]float64

	W := cal.CalibrationWeight
	// Build measurement rows: order cell0..cell3, center
	measurements := [5][4]float64{
		cal.OnCell0,
		cal.OnCell1,
		cal.OnCell2,
		cal.OnCell3,
		cal.OnCenter,
	}

	// Build X (5x4) and y (5)
	const m = 5
	var X [m][4]float64
	var y [m]float64
	for i := 0; i < m; i++ {
		for j := 0; j < 4; j++ {
			X[i][j] = measurements[i][j] - cal.Zero[j]
		}
		// For each placement the observed weight is W
		y[i] = W
	}

	// Compute normal matrix A = X^T X (4x4) and b = X^T y (4)
	var A [4][4]float64
	var b [4]float64
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			sum := 0.0
			for k := 0; k < m; k++ {
				sum += X[k][i] * X[k][j]
			}
			A[i][j] = sum
		}
		sum := 0.0
		for k := 0; k < m; k++ {
			sum += X[k][i] * y[k]
		}
		b[i] = sum
	}

	// Solve A f = b
	sol, err := solve4x4(A, b)
	if err != nil {
		return factors, fmt.Errorf("could not solve normal equations: %w", err)
	}
	for i := 0; i < 4; i++ {
		factors[i] = sol[i]
	}
	return factors, nil
}

// ComputeWeight computes the estimated actual weight for a 4-channel ADC reading given zero reference and factors.
func ComputeWeight(adc [4]float64, zero [4]float64, factors [4]float64) float64 {
	w := 0.0
	for i := 0; i < 4; i++ {
		w += factors[i] * (adc[i] - zero[i])
	}
	return w
}

// solve4x4 solves A x = b for 4x4 A and length-4 b using Gaussian elimination with partial pivoting.
// Returns error if matrix is singular.
func solve4x4(A [4][4]float64, b [4]float64) ([4]float64, error) {
	var aug [4][5]float64
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			aug[i][j] = A[i][j]
		}
		aug[i][4] = b[i]
	}

	// Forward elimination with partial pivoting
	for col := 0; col < 4; col++ {
		// find pivot
		pivot := col
		maxAbs := abs(aug[col][col])
		for r := col + 1; r < 4; r++ {
			if abs(aug[r][col]) > maxAbs {
				maxAbs = abs(aug[r][col])
				pivot = r
			}
		}
		if maxAbs == 0 {
			return [4]float64{}, errors.New("matrix is singular (zero pivot)")
		}
		// swap rows if needed
		if pivot != col {
			aug[col], aug[pivot] = aug[pivot], aug[col]
		}
		// normalize and eliminate below
		for r := col + 1; r < 4; r++ {
			factor := aug[r][col] / aug[col][col]
			for c := col; c < 5; c++ {
				aug[r][c] -= factor * aug[col][c]
			}
		}
	}

	// Back substitution
	var x [4]float64
	for i := 3; i >= 0; i-- {
		if aug[i][i] == 0 {
			return [4]float64{}, errors.New("singular matrix during back substitution")
		}
		sum := aug[i][4]
		for j := i + 1; j < 4; j++ {
			sum -= aug[i][j] * x[j]
		}
		x[i] = sum / aug[i][i]
	}
	return x, nil
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
