package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	calPath := flag.String("cal", "calibration.json", "path to calibration JSON (required)")
	adcStr := flag.String("adc", "", "comma-separated 4 ADC values to compute weight, e.g. 1020,1018,1005,1009")
	adcFile := flag.String("adc-file", "", "path to JSON file containing an array of adc readings or single adc")
	apply := flag.Bool("apply", false, "when set, process ADC inputs; otherwise only run verification")
	jsonOut := flag.String("json-out", "", "write results to this JSON file")
	flag.Parse()

	if calPath == nil || *calPath == "" {
		fmt.Fprintln(os.Stderr, "error: -cal is required")
		flag.Usage()
		os.Exit(2)
	}

	dataBytes, err := os.ReadFile(*calPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading calibration file: %v\n", err)
		os.Exit(1)
	}

	var cal CalibrationData
	if err := json.Unmarshal(dataBytes, &cal); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing calibration JSON: %v\n", err)
		os.Exit(1)
	}

	// read optional ridge regularization and print-normal flags from environment
	ridge := 0.0
	if rv := os.Getenv("CAL_RIDGE"); rv != "" {
		if v, err := strconv.ParseFloat(rv, 64); err == nil {
			ridge = v
		}
	}
	printNormal := false
	if pv := os.Getenv("CAL_PRINT_NORMAL"); pv == "1" || strings.ToLower(pv) == "true" {
		printNormal = true
	}

	// Parse ADC input (single or array) early so flags are validated but we only process when -apply is set
	var adcInput [4]float64
	haveADC := false
	var manyReadings [][]float64
	if *adcStr != "" {
		parts := strings.Split(*adcStr, ",")
		if len(parts) != 4 {
			fmt.Fprintln(os.Stderr, "error: -adc must have 4 comma-separated values")
			os.Exit(2)
		}
		for i := 0; i < 4; i++ {
			v, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing adc value %q: %v\n", parts[i], err)
				os.Exit(1)
			}
			adcInput[i] = v
		}
		haveADC = true
	} else if *adcFile != "" {
		b, err := os.ReadFile(*adcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading adc file: %v\n", err)
			os.Exit(1)
		}
		// try single object {"adc": [..]}
		var single struct {
			ADC [4]float64 `json:"adc"`
		}
		if err := json.Unmarshal(b, &single); err == nil && (single.ADC != [4]float64{}) {
			adcInput = single.ADC
			haveADC = true
		} else {
			// try raw array of arrays
			var many [][]float64
			if err := json.Unmarshal(b, &many); err == nil && len(many) > 0 {
				manyReadings = many
				// use first as adcInput
				if len(many[0]) != 4 {
					fmt.Fprintln(os.Stderr, "error: each adc reading must have 4 values")
					os.Exit(2)
				}
				for i := 0; i < 4; i++ {
					adcInput[i] = many[0][i]
				}
				haveADC = true
			} else {
				// try object {"adc": [[..],[..]]}
				var objMany struct {
					ADC [][]float64 `json:"adc"`
				}
				if err := json.Unmarshal(b, &objMany); err == nil && len(objMany.ADC) > 0 {
					manyReadings = objMany.ADC
					if len(objMany.ADC[0]) != 4 {
						fmt.Fprintln(os.Stderr, "error: each adc reading must have 4 values")
						os.Exit(2)
					}
					for i := 0; i < 4; i++ {
						adcInput[i] = objMany.ADC[0][i]
					}
					haveADC = true
				} else {
					fmt.Fprintf(os.Stderr, "error parsing adc file: unsupported format\n")
					os.Exit(1)
				}
			}
		}
	}

	factors, A, b, err := ComputeFactors(cal, ridge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calculation error: %v\n", err)
		os.Exit(1)
	}
	if printNormal {
		fmt.Println("Normal matrix A:")
		for i := 0; i < 4; i++ {
			fmt.Println(A[i])
		}
		fmt.Println("Right-hand side b:")
		fmt.Println(b)
	}

	// Header
	fmt.Printf("Calibration weight W = %g\n", cal.CalibrationWeight)
	fmt.Println("Zero reference (adc):", cal.Zero)
	fmt.Printf("Computed factors f0..f3 (weight per ADC count):\n")
	for i, f := range factors {
		fmt.Printf("  f%d = %.10g\n", i, f)
	}

	// Verification using calibration rows (no extra file):
	calibRows := [][]float64{
		{cal.OnCell0[0], cal.OnCell0[1], cal.OnCell0[2], cal.OnCell0[3]},
		{cal.OnCell1[0], cal.OnCell1[1], cal.OnCell1[2], cal.OnCell1[3]},
		{cal.OnCell2[0], cal.OnCell2[1], cal.OnCell2[2], cal.OnCell2[3]},
		{cal.OnCell3[0], cal.OnCell3[1], cal.OnCell3[2], cal.OnCell3[3]},
		{cal.OnCenter[0], cal.OnCenter[1], cal.OnCenter[2], cal.OnCenter[3]},
	}
	fmt.Println("\nVerification using calibration ADC rows:")
	for idx, row := range calibRows {
		var adr [4]float64
		for i := 0; i < 4; i++ {
			adr[i] = row[i]
		}
		var delta [4]float64
		var contrib [4]float64
		for i := 0; i < 4; i++ {
			delta[i] = adr[i] - cal.Zero[i]
			contrib[i] = factors[i] * delta[i]
		}
		weight := 0.0
		for i := 0; i < 4; i++ {
			weight += contrib[i]
		}
		fmt.Printf("Row %d ADC=%v\n", idx+1, adr)
		fmt.Printf("  Delta: %v\n", delta)
		// print Contrib with two decimals
		fmt.Printf("  Contrib: [%.2f %.2f %.2f %.2f]\n", contrib[0], contrib[1], contrib[2], contrib[3])
		fmt.Printf("  Estimated weight = %.2f (expected %.2f)\n\n", weight, cal.CalibrationWeight)
	}

	// Compute residuals and an "error determinant" metric: det(A) * residualVariance
	// residualVariance = RSS / (m - p) where m=5 rows, p=4 parameters
	m := len(calibRows)
	var rss float64
	for _, row := range calibRows {
		var adr [4]float64
		for i := 0; i < 4; i++ {
			adr[i] = row[i]
		}
		est := 0.0
		for i := 0; i < 4; i++ {
			est += factors[i] * (adr[i] - cal.Zero[i])
		}
		resid := cal.CalibrationWeight - est
		rss += resid * resid
	}
	df := float64(m - 4)
	var residualVar float64
	if df > 0 {
		residualVar = rss / df
	} else {
		residualVar = rss
	}
	detA := det4x4(A)
	errorDet := detA * residualVar
	fmt.Printf("Residual variance = %.6g (RSS=%.6g, df=%v)\n", residualVar, rss, int(df))
	fmt.Printf("det(A) = %.6g\n", detA)
	fmt.Printf("error determinant (det(A) * residualVariance) = %.6g\n", errorDet)

	// Prepare output buffer and write header
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Calibration weight W = %g\n", cal.CalibrationWeight))
	sb.WriteString(fmt.Sprintf("Zero reference (adc): %v\n", cal.Zero))
	sb.WriteString("Computed factors f0..f3 (weight per ADC count):\n")
	for i, f := range factors {
		sb.WriteString(fmt.Sprintf("  f%d = %.10g\n", i, f))
	}

	// Process ADC input(s) only if -apply is set
	if *apply && haveADC {
		if len(manyReadings) > 0 {
			for idx, row := range manyReadings {
				if len(row) != 4 {
					continue
				}
				var adr [4]float64
				for i := 0; i < 4; i++ {
					adr[i] = row[i]
				}
				var delta [4]float64
				var contrib [4]float64
				for i := 0; i < 4; i++ {
					delta[i] = adr[i] - cal.Zero[i]
					contrib[i] = factors[i] * delta[i]
				}
				weight := 0.0
				for i := 0; i < 4; i++ {
					weight += contrib[i]
				}
				fmt.Printf("Reading %d: ADC=%v\n", idx+1, adr)
				fmt.Printf("  Delta: %v\n", delta)
				// print Contrib with two decimals
				fmt.Printf("  Contrib: [%.2f %.2f %.2f %.2f]\n", contrib[0], contrib[1], contrib[2], contrib[3])
				fmt.Printf("  Estimated weight = %.2f\n", weight)
				sb.WriteString(fmt.Sprintf("\nReading %d: ADC=%v\n", idx+1, adr))
				sb.WriteString(fmt.Sprintf("  Delta: %v\n", delta))
				sb.WriteString(fmt.Sprintf("  Contrib: [%.2f %.2f %.2f %.2f]\n", contrib[0], contrib[1], contrib[2], contrib[3]))
				sb.WriteString(fmt.Sprintf("  Estimated weight = %.2f\n", weight))
			}
		} else {
			var delta [4]float64
			var contrib [4]float64
			for i := 0; i < 4; i++ {
				delta[i] = adcInput[i] - cal.Zero[i]
				contrib[i] = factors[i] * delta[i]
			}
			weight := 0.0
			for i := 0; i < 4; i++ {
				weight += contrib[i]
			}
			fmt.Printf("Input ADC: %v\n", adcInput)
			fmt.Printf("  Delta: %v\n", delta)
			fmt.Printf("  Contrib: [%.2f %.2f %.2f %.2f]\n", contrib[0], contrib[1], contrib[2], contrib[3])
			fmt.Printf("  Estimated weight = %.2f (same units as calibration weight)\n", weight)
			sb.WriteString(fmt.Sprintf("Input ADC: %v\n", adcInput))
			sb.WriteString(fmt.Sprintf("  Delta: %v\n", delta))
			sb.WriteString(fmt.Sprintf("  Contrib: [%.2f %.2f %.2f %.2f]\n", contrib[0], contrib[1], contrib[2], contrib[3]))
			sb.WriteString(fmt.Sprintf("  Estimated weight = %.2f (same units as calibration weight)\n", weight))
		}
	}

	// If no JSON output is requested, write the human-readable output.txt
	if *jsonOut == "" {
		_ = os.WriteFile("output.txt", []byte(sb.String()), 0644)
	}

	// If requested, write a JSON summary (and skip text output when set)
	if *jsonOut != "" {
		res := CalibrationResult{
			Factors:       factors,
			ResidualVar:   residualVar,
			RSS:           rss,
			DetA:          detA,
			ErrorDet:      errorDet,
			CalibrationW:  cal.CalibrationWeight,
			CalibrationOK: residualVar < 1e-6,
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		_ = os.WriteFile(*jsonOut, out, 0644)
	}
}
