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

// CalibrationResult is the JSON schema written when -json-out is used.
type CalibrationResult struct {
	Factors       [4]float64 `json:"factors"`
	ResidualVar   float64    `json:"residual_variance"`
	RSS           float64    `json:"rss"`
	DetA          float64    `json:"det_A"`
	ErrorDet      float64    `json:"error_det"`
	CalibrationW  float64    `json:"calibration_weight"`
	CalibrationOK bool       `json:"calibration_ok"`
}
