```markdown
# Calibration-Demo — 4-Loadcell Least Squares Calibration (Go)

This tool reads a calibration JSON, computes least-squares scale factors for 4 load cells, and estimates weight from ADC readings.

Quick start:
1. Build:
   go build -o calibrate

2. Example:
   ./calibrate -cal calibration-example.json
   ./calibrate -cal calibration-example.json -adc "1020,1018,1005,1009"
   ./calibrate -cal calibration-example.json -adc-file adc-input.json

JSON schema (see calibration-example.json):
{
  "calibration_weight": 100.0,
  "zero": [z0,z1,z2,z3],
  "on_cell_0": [adc0,adc1,adc2,adc3],
  "on_cell_1": [...],
  "on_cell_2": [...],
  "on_cell_3": [...],
  "on_center": [...]
}

Notes:
- The solver forms normal equations (X^T X) f = X^T y where each row of X is (adc - zero) for the five measurements (cell0..cell3 and center).
- If the normal matrix is singular (insufficient independent measurements), the solver will return an error. You may add more measurement positions to improve robustness.
```