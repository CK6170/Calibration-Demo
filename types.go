package main

// CalibrationData defines the expected JSON schema for calibration input.
type CalibrationData struct {
	CalibrationWeight float64    `json:"calibration_weight"`
	Zero              [4]float64 `json:"zero"`
	OnCell0           [4]float64 `json:"on_cell_0"`
	OnCell1           [4]float64 `json:"on_cell_1"`
	OnCell2           [4]float64 `json:"on_cell_2"`
	OnCell3           [4]float64 `json:"on_cell_3"`
	OnCenter          [4]float64 `json:"on_center"`
}
