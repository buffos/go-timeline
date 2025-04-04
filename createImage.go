// createImage.go
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	// "image" // No longer needed unless doing JPG conversion
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"strings"

	// "math" // No longer needed

	"github.com/chromedp/chromedp"
)

// Removed const defaultImageWidth/Height - determined from SVG by browser now
// Removed const defaultResolution - handled by screenshot

// Update function signature - remove outputFilename
func generateImage(template Template, entries []TimelineEntry, format string, outputWriter io.Writer) error {
	// 1. Generate SVG string first
	svgString, err := GenerateSVG(template, entries)
	if err != nil {
		return fmt.Errorf("failed to generate intermediate SVG: %w", err)
	}

	// --- Use chromedp to render SVG ---

	// 2. Create a base64 data URI for the SVG
	// This allows loading the SVG directly without saving a temp file
	svgBase64 := base64.StdEncoding.EncodeToString([]byte(svgString))
	dataURI := "data:image/svg+xml;base64," + svgBase64
	log.Println("Created data URI for SVG.")

	// 3. Setup chromedp
	// Create allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// Add options here if needed, e.g.:
		// chromedp.DisableGPU,
		// chromedp.NoSandbox,
		chromedp.Headless, // Ensure it runs headless
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	// Create a new context
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// 4. Define tasks to navigate and screenshot the SVG element
	var screenshotBuf []byte

	tasks := chromedp.Tasks{
		// Navigate to the data URI
		chromedp.Navigate(dataURI),
		// Wait for the svg element to be present
		chromedp.WaitVisible(`svg`, chromedp.ByQuery),
		// Take a screenshot of the first SVG element found
		chromedp.Screenshot(`svg`, &screenshotBuf, chromedp.ByQuery),
	}

	// 5. Run the tasks
	log.Println("Running chromedp tasks (navigate and screenshot)...")
	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("chromedp execution failed: %w", err)
	}
	log.Println("Chromedp tasks completed successfully.")

	if len(screenshotBuf) == 0 {
		return fmt.Errorf("screenshot buffer is empty, screenshot failed")
	}

	// 6. Process output
	screenshotReader := bytes.NewReader(screenshotBuf)

	switch format {
	case "png":
		// Screenshot is already PNG, just copy it
		_, err = io.Copy(outputWriter, screenshotReader)
		if err != nil {
			return fmt.Errorf("failed to write PNG screenshot data: %w", err)
		}
	case "jpg", "jpeg":
		// Decode the PNG screenshot
		img, errPng := png.Decode(screenshotReader) // Use different error var name
		if errPng != nil {
			return fmt.Errorf("failed to decode PNG screenshot: %w", errPng)
		}
		// Re-encode as JPEG
		opts := &jpeg.Options{Quality: 90} // Default JPEG quality
		err = jpeg.Encode(outputWriter, img, opts)
		if err != nil {
			return fmt.Errorf("failed to encode JPEG: %w", err)
		}
	default:
		return fmt.Errorf("internal error: unsupported image format '%s' with chromedp", format)
	}

	log.Printf("Successfully encoded %s image using chromedp.", strings.ToUpper(format))
	return nil
}
