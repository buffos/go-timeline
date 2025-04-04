// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log" // Needed for rounding rect dimensions
	"os"
	"strings"
)

// --- Main Program Logic ---

func main() { // NOSONAR
	// --- Setup Fonts ---
	// setupFonts() // REMOVED

	// --- Argument Parsing using flag package ---
	outputFile := flag.String("o", "", "Output file path (default: stdout)")
	// Add other flags here if needed in the future
	flag.Parse() // Parse the flags provided

	// Get positional arguments (template, data, format) after flags
	args := flag.Args()
	if len(args) != 3 {
		// Improved usage message
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <template.json> <data.json> <format>\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nArguments:")
		fmt.Fprintln(os.Stderr, "  <template.json>   Path to the template definition file.")
		fmt.Fprintln(os.Stderr, "  <data.json>       Path to the timeline data file.")
		fmt.Fprintln(os.Stderr, "  <format>          Output format (svg, html, png, jpg/jpeg).")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults() // Print default flag values and descriptions
		os.Exit(1)           // Exit with error code
	}
	templateFile := args[0]
	dataFile := args[1]
	exportFormat := strings.ToLower(args[2])

	// --- File Reading & Parsing ---
	log.Printf("Reading template file: %s", templateFile)
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		log.Fatalf("Error reading template file '%s': %v", templateFile, err)
	}
	log.Printf("Reading data file: %s", dataFile)
	dataBytes, err := os.ReadFile(dataFile)
	if err != nil {
		log.Fatalf("Error reading data file '%s': %v", dataFile, err)
	}

	var template Template
	log.Println("Parsing template JSON...")
	err = json.Unmarshal(templateBytes, &template)
	if err != nil {
		log.Fatalf("Error parsing template JSON '%s': %v", templateFile, err)
	}

	var timelineData TimelineData
	log.Println("Parsing data JSON...")
	// Attempt parsing as {"entries": [...]} first
	err = json.Unmarshal(dataBytes, &timelineData)
	if err != nil {
		// Fallback: Try parsing directly as an array [...]
		log.Printf("Warning: Failed to parse data as root object ('%v'), attempting direct array parsing.", err)
		var entriesDirect []TimelineEntry
		errDirect := json.Unmarshal(dataBytes, &entriesDirect)
		if errDirect != nil {
			// Report the *original* error, as it's more likely the intended format failed
			log.Fatalf("Error parsing data JSON '%s': %v (also failed direct array parse: %v)", dataFile, err, errDirect)
		}
		timelineData.Entries = entriesDirect
		log.Println("Successfully parsed data JSON as a direct array.")
	} else {
		log.Println("Successfully parsed data JSON with 'entries' root key.")
	}

	// --- Input Validation ---
	log.Println("Validating inputs...")
	supportedFormats := map[string]bool{"html": true, "svg": true, "png": true, "jpg": true, "jpeg": true}
	if !supportedFormats[exportFormat] {
		log.Fatalf("Unsupported export format '%s'. Supported formats: html, svg, png, jpg/jpeg", exportFormat)
	}
	if template.CenterLine.Orientation != "horizontal" && template.CenterLine.Orientation != "vertical" {
		log.Fatalf("Template error: center_line.orientation must be 'horizontal' or 'vertical'")
	}
	if len(timelineData.Entries) == 0 {
		log.Fatalf("Data error: No timeline entries found in '%s'", dataFile)
	}
	log.Println("Inputs validated successfully.")

	// --- Determine Output Writer ---
	var outputWriter io.Writer = os.Stdout // Default to standard output
	var outFile *os.File = nil             // Keep track of the file if opened

	if *outputFile != "" {
		log.Printf("Output directed to file: %s", *outputFile)
		outFile, err = os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Error creating output file '%s': %v", *outputFile, err)
		}
		// Use defer to ensure file is closed, even on panic (though we handle errors before)
		defer func() {
			if outFile != nil {
				log.Printf("Closing output file: %s", *outputFile)
				closeErr := outFile.Close()
				if closeErr != nil {
					// Log error but don't override fatal errors from generation
					log.Printf("Error closing output file '%s': %v", *outputFile, closeErr)
				}
			}
		}()
		outputWriter = outFile // Set writer to the file
	} else {
		log.Println("Output directed to stdout.")
	}

	// --- Generation ---
	log.Printf("Generating output for format: %s", exportFormat)
	var genErr error

	switch exportFormat {
	case "svg":
		svgContent, errSvg := GenerateSVG(template, timelineData.Entries)
		if errSvg != nil {
			genErr = fmt.Errorf("SVG generation failed: %w", errSvg)
		} else {
			_, genErr = io.WriteString(outputWriter, svgContent) // Write string directly
			if genErr != nil {
				genErr = fmt.Errorf("failed to write SVG output: %w", genErr)
			}
		}
	case "html":
		outputString, errHtml := generateHTML(template, timelineData.Entries)
		if errHtml != nil {
			genErr = fmt.Errorf("HTML generation failed: %w", errHtml)
		} else {
			_, genErr = io.WriteString(outputWriter, outputString) // Write string directly
			if genErr != nil {
				genErr = fmt.Errorf("failed to write HTML output: %w", genErr)
			}
		}
	case "png", "jpg", "jpeg":
		// Call the image generation function, passing the determined writer
		genErr = generateImage(template, timelineData.Entries, exportFormat, outputWriter)
		// Error wrapping happens within generateImage if needed
	}

	// --- Handle Generation Errors ---
	if genErr != nil {
		log.Fatalf("Error generating %s: %v", exportFormat, genErr)
		// Note: Defer will close the file if it was opened.
		// We could attempt to remove the potentially partial file here, but defer handles closure.
		// If writing failed mid-stream, the file might be partial.
		// If generation failed *before* writing, the file might be empty or non-existent.
		if outFile != nil && *outputFile != "" {
			// Attempt cleanup if error occurred and we were writing to a file
			log.Printf("Attempting to remove potentially incomplete file: %s", *outputFile)
			// Ensure file is closed *before* removing (defer will handle this, but being explicit can help reasoning)
			// outFile.Close() // Defer handles this
			removeErr := os.Remove(*outputFile)
			if removeErr != nil {
				log.Printf("Warning: Could not remove output file '%s' after error: %v", *outputFile, removeErr)
			}
		}
		// No need to os.Exit(1) here, log.Fatalf already does that.
	} else {
		log.Printf("Successfully generated %s output.", strings.ToUpper(exportFormat))
		if *outputFile != "" {
			log.Printf("Output saved to: %s", *outputFile)
		}
	}
}
