package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func main() {
	calPath := flag.String("cal", "calibration.json", "path to calibration JSON (required)")
	adcStr := flag.String("adc", "", "comma-separated 4 ADC values to compute weight, e.g. 1020,1018,1005,1009")
	adcFile := flag.String("adc-file", "", "path to JSON file containing {\"adc\": [v0,v1,v2,v3]} to compute weight")
	flag.Parse()

	if calPath == nil || *calPath == "" {
		fmt.Fprintln(os.Stderr, "error: -cal is required")
		flag.Usage()
		os.Exit(2)
	}

	dataBytes, err := ioutil.ReadFile(*calPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading calibration file: %v\n", err)
		os.Exit(1)
	}

	var cal CalibrationData
	if err := json.Unmarshal(dataBytes, &cal); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing calibration JSON: %v\n", err)
		os.Exit(1)
	}

	factors, err := ComputeFactors(cal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calculation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Calibration weight W = %g\n", cal.CalibrationWeight)
	fmt.Println("Zero reference (adc):", cal.Zero)
	fmt.Printf("Computed factors f0..f3 (weight per ADC count):\n")
	for i, f := range factors {
		fmt.Printf("  f%d = %.10g\n", i, f)
	}

	// If ADC input requested
	var adcInput [4]float64
	haveADC := false
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
		b, err := ioutil.ReadFile(*adcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading adc file: %v\n", err)
			os.Exit(1)
		}
		var obj struct {
			ADC [4]float64 `json:"adc"`
		}
		if err := json.Unmarshal(b, &obj); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing adc file: %v\n", err)
			os.Exit(1)
		}
		adcInput = obj.ADC
		haveADC = true
	}

	if haveADC {
		weight := ComputeWeight(adcInput, cal.Zero, factors)
		fmt.Printf("Input ADC: %v\n", adcInput)
		fmt.Printf("Estimated weight = %.6g (same units as calibration weight)\n", weight)
	}
}
