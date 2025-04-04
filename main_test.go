package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSVGGeneration performs SVG comparison testing.
func TestSVGGeneration(t *testing.T) {
	testDataDir := "testdata"

	// Find all test template files
	templateFiles, err := filepath.Glob(filepath.Join(testDataDir, "*.tmpl.json"))
	if err != nil {
		t.Fatalf("Error finding template files: %v", err)
	}
	if len(templateFiles) == 0 {
		t.Fatalf("No template files found in %s", testDataDir)
	}

	for _, templateFile := range templateFiles {
		baseName := strings.TrimSuffix(filepath.Base(templateFile), ".tmpl.json")
		t.Run(baseName, func(t *testing.T) {
			dataFile := filepath.Join(testDataDir, baseName+".data.json")
			expectedSVGFile := filepath.Join(testDataDir, baseName+".expected.svg")

			// --- Load Template ---
			templateBytes, err := os.ReadFile(templateFile)
			if err != nil {
				t.Fatalf("Error reading template file %s: %v", templateFile, err)
			}
			var template Template
			if err := json.Unmarshal(templateBytes, &template); err != nil {
				t.Fatalf("Error unmarshalling template %s: %v", templateFile, err)
			}

			// --- Load Data ---
			dataBytes, err := os.ReadFile(dataFile)
			if err != nil {
				t.Fatalf("Error reading data file %s: %v", dataFile, err)
			}
			var data TimelineData // Assuming your data structure is named this
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				t.Fatalf("Error unmarshalling data %s: %v", dataFile, err)
			}

			// --- Generate SVG ---
			generatedSVG, err := GenerateSVG(template, data.Entries) // Use the correct field name for entries
			if err != nil {
				t.Fatalf("Error generating SVG for %s: %v", baseName, err)
			}

			// --- Load Expected SVG ---
			expectedSVGBytes, err := os.ReadFile(expectedSVGFile)
			if err != nil {
				// If the expected file doesn't exist, maybe create it?
				if os.IsNotExist(err) {
					t.Logf("Expected SVG file %s not found. Creating it.", expectedSVGFile)
					if writeErr := os.WriteFile(expectedSVGFile, []byte(generatedSVG), 0644); writeErr != nil {
						t.Errorf("Failed to write new expected SVG %s: %v", expectedSVGFile, writeErr)
					}
					// Continue to the next test after creating the initial snapshot
					return
				}
				t.Fatalf("Error reading expected SVG file %s: %v", expectedSVGFile, err)
			}
			expectedSVG := string(expectedSVGBytes)

			// --- Compare SVG ---
			// Normalize line endings for comparison
			normalizedGenerated := strings.ReplaceAll(generatedSVG, "\r\n", "\n")
			normalizedExpected := strings.ReplaceAll(expectedSVG, "\r\n", "\n")

			if normalizedGenerated != normalizedExpected {
				diff := findFirstDifference(normalizedGenerated, normalizedExpected)
				t.Errorf("Generated SVG for %s does not match %s.\nFirst difference near character %d:\nEXPECTED:\n...%s...\nGOT:\n...%s...",
					baseName, expectedSVGFile,
					diff.Index, diff.ExpectedContext, diff.GotContext)
				// Optional: Write the failed output for easier comparison
				failedFile := filepath.Join(testDataDir, baseName+".failed.svg")
				os.WriteFile(failedFile, []byte(generatedSVG), 0644)
				t.Logf("Wrote differing output to %s", failedFile)
			}
		})
	}
}

// diffResult helps show context around the first difference.
type diffResult struct {
	Index           int
	ExpectedContext string
	GotContext      string
}

// findFirstDifference finds the first differing character and provides context.
func findFirstDifference(s1, s2 string) diffResult {
	limit := len(s1)
	if len(s2) < limit {
		limit = len(s2)
	}
	idx := -1
	for i := 0; i < limit; i++ {
		if s1[i] != s2[i] {
			idx = i
			break
		}
	}
	// Handle case where one string is a prefix of the other
	if idx == -1 && len(s1) != len(s2) {
		idx = limit
	}
	if idx == -1 { // Should not happen if strings are different, but handle gracefully
		return diffResult{Index: 0, ExpectedContext: "(Strings are identical)", GotContext: "(Strings are identical)"}
	}

	contextSize := 20 // Characters before and after the difference
	start := idx - contextSize
	if start < 0 {
		start = 0
	}
	endS1 := idx + contextSize
	if endS1 > len(s1) {
		endS1 = len(s1)
	}
	endS2 := idx + contextSize
	if endS2 > len(s2) {
		endS2 = len(s2)
	}

	return diffResult{
		Index:           idx,
		ExpectedContext: s1[start:endS1],
		GotContext:      s2[start:endS2],
	}
}
