package main

import (
	"fmt"
	"strconv"
	"strings"
)

// --- Helper Functions for Effective Styles ---

// Helper to get value from pointer or default
func getString(ptr *string, def string) string {
	if ptr != nil {
		return *ptr
	}
	return def
}
func getInt(ptr *int, def int) int {
	if ptr != nil {
		return *ptr
	}
	return def
}
func getFloat64(ptr *float64, def float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return def
}
func getBool(ptr *bool, def bool) bool {
	if ptr != nil {
		return *ptr
	}
	return def
}

func getEffectiveJunctionMarkerStyle(defaults JunctionMarkerStyle, override *JunctionMarkerOverride) JunctionMarkerStyle {
	if override == nil {
		return defaults
	}
	effective := defaults
	effective.Shape = getString(override.Shape, defaults.Shape)
	effective.Size = getFloat64(override.Size, defaults.Size)
	// Color is tricky - override takes precedence, then default field, then it's usually derived
	if override.Color != nil {
		effective.Color = override.Color // Pointer allows explicit override
	} // Keep default struct Color pointer if override doesn't specify
	return effective
}

func getEffectiveTitleLineStyle(defaults TitleLineStyle, override *TitleLineStyleOverride) TitleLineStyle {
	if override == nil {
		return defaults
	}
	effective := defaults
	effective.Visible = getBool(override.Visible, defaults.Visible) // Assumes default has sensible Visible value
	effective.Color = getString(override.Color, defaults.Color)
	effective.Width = getFloat64(override.Width, defaults.Width)
	effective.Length = getFloat64(override.Length, defaults.Length)
	effective.Margin = getFloat64(override.Margin, defaults.Margin)

	// Re-evaluate visibility based on dimensions if not explicitly set by override
	if override.Visible == nil { // If visibility wasn't overridden
		if effective.Width > 0 && effective.Length > 0 {
			effective.Visible = true
		} else if defaults.Visible { // Inherit default visibility only if dims are zero
			effective.Visible = true
		} else {
			effective.Visible = false
		}
	}

	return effective
}

// Helper to merge DotStyle with DotStyleOverride
func getEffectiveDotStyle(defaults DotStyle, override *DotStyleOverride) DotStyle {
	if override == nil {
		return defaults
	}
	effective := defaults // Start with defaults
	effective.Size = getInt(override.Size, defaults.Size)
	effective.Color = getString(override.Color, defaults.Color)
	effective.Shape = getString(override.Shape, defaults.Shape)
	effective.Visible = getBool(override.Visible, true)
	effective.OffsetMain = getInt(override.OffsetMain, defaults.OffsetMain)
	effective.OffsetCross = getInt(override.OffsetCross, defaults.OffsetCross)
	// Default stop_at_dot to true if not overridden
	effective.StopAtDot = getBool(override.StopAtDot, true)
	return effective
}

// (getEffectiveConnectorStyle - Needs update if Dot becomes pointer or has complex merge)
// Assuming Dot merge logic from previous step is sufficient for now. Needs review if granular Dot override is needed.
func getEffectiveConnectorStyle(defaults ConnectorStyle, override *ConnectorStyleOverride) ConnectorStyle {
	if override == nil {
		return defaults
	}
	effective := defaults
	// Use helper functions for pointer overrides
	effective.Color = getString(override.Color, defaults.Color)
	effective.LineType = getString(override.LineType, defaults.LineType)
	effective.Width = getInt(override.Width, defaults.Width)
	// Use getBool to merge the flags, providing a default value (true)
	defaultDrawToPeriod := true
	if defaults.DrawToPeriod != nil { // If default struct has a non-nil value, use it
		defaultDrawToPeriod = *defaults.DrawToPeriod
	}
	drawToPeriodResult := getBool(override.DrawToPeriod, defaultDrawToPeriod)
	effective.DrawToPeriod = &drawToPeriodResult // Store the final result as a pointer

	defaultDrawToComment := true
	if defaults.DrawToComment != nil { // If default struct has a non-nil value, use it
		defaultDrawToComment = *defaults.DrawToComment
	}
	drawToCommentResult := getBool(override.DrawToComment, defaultDrawToComment)
	effective.DrawToComment = &drawToCommentResult // Store the final result as a pointer

	// Merge Dot style using the new helper
	effective.Dot = getEffectiveDotStyle(defaults.Dot, override.Dot)

	// Dot Merge Logic (Needs review/update if DotStyle itself uses pointers or needs granular override)
	// Current logic assumes override Dot is not a pointer and replaces if present.
	// If override struct *had* a Dot *DotStyleOverride field:
	/*
		if override.Dot != nil {
			dotDefaults := defaults.Dot
			effectiveDot := defaults.Dot // Start with default dot
			// Merge override.Dot fields into effectiveDot using get... helpers
			effectiveDot.Size = getInt(override.Dot.Size, dotDefaults.Size) // Assuming DotOverride had *int, etc.
			effectiveDot.Color = getString(override.Dot.Color, dotDefaults.Color)
			effectiveDot.Shape = getString(override.Dot.Shape, dotDefaults.Shape)
			effectiveDot.Visible = getBool(override.Dot.Visible, dotDefaults.Visible)
			effectiveDot.Offset = getInt(override.Dot.Offset, dotDefaults.Offset)
			effective.Dot = effectiveDot
		}
	*/
	// Since override.Dot is not currently part of ConnectorStyleOverride, this block is commented out.
	// The original simple Dot merge logic (if any existed before) might need restoration
	// or a proper DotStyleOverride needs to be added to models.go if needed.

	return effective
}

func getEffectiveCommentTextStyle(globalFont *FontStyle, defaults CommentTextStyle, override *CommentTextStyleOverride) CommentTextStyle {
	// --- DEBUG LOGGING START ---
	// log.Printf("DEBUG: Entering getEffectiveCommentTextStyle. Override provided: %t", override != nil)
	// if override != nil {
	// 	log.Printf("DEBUG: Checking override.MainAxisOffset. Is nil? %t", override.MainAxisOffset == nil)
	// 	if override.MainAxisOffset != nil {
	// 		log.Printf("DEBUG: override.MainAxisOffset value = %.2f", *override.MainAxisOffset)
	// 	}
	// 	log.Printf("DEBUG: Checking override.CrossAxisOffset. Is nil? %t", override.CrossAxisOffset == nil)
	// 	if override.CrossAxisOffset != nil {
	// 		log.Printf("DEBUG: override.CrossAxisOffset value = %.2f", *override.CrossAxisOffset)
	// 	}
	// }
	// log.Printf("DEBUG: Default MainAxisOffset=%.2f, CrossAxisOffset=%.2f", defaults.MainAxisOffset, defaults.CrossAxisOffset)
	// --- DEBUG LOGGING END ---

	effective := defaults
	bodyFontOverride := (*FontStyleOverride)(nil)
	titleFontOverride := (*FontStyleOverride)(nil)
	titleLineOverride := (*TitleLineStyleOverride)(nil)

	if override != nil {
		effective.Position = getString(override.Position, defaults.Position)
		effective.MainAxisOffset = getFloat64(override.MainAxisOffset, defaults.MainAxisOffset) // Tries to apply override
		effective.CrossAxisOffset = getFloat64(override.CrossAxisOffset, defaults.CrossAxisOffset)
		effective.TitleColor = getString(override.TitleColor, defaults.TitleColor)
		effective.Shape = getString(override.Shape, defaults.Shape)
		effective.FillColor = getString(override.FillColor, defaults.FillColor)
		effective.TextColor = getString(override.TextColor, defaults.TextColor)
		effective.Padding = getString(override.Padding, defaults.Padding)
		effective.BlockWidth = override.BlockWidth // Directly assign pointer; nil if not overridden
		effective.BorderColor = getString(override.BorderColor, defaults.BorderColor)
		effective.BorderWidth = getInt(override.BorderWidth, defaults.BorderWidth)
		effective.BorderStyle = getString(override.BorderStyle, defaults.BorderStyle)
		effective.TextAlign = getString(override.TextAlign, defaults.TextAlign)
		bodyFontOverride = override.Font
		titleFontOverride = override.TitleFont
		titleLineOverride = override.TitleLine
	}

	// Merge BlockWidth (if override didn't set it, keep default's pointer)
	if effective.BlockWidth == nil {
		effective.BlockWidth = defaults.BlockWidth
	}

	// Get effective font styles
	effective.Font = getEffectiveFontStyle(globalFont, defaults.Font, bodyFontOverride)
	effective.TitleFont = getEffectiveFontStyle(globalFont, defaults.TitleFont, titleFontOverride)
	// Get effective title line style
	effective.TitleLine = getEffectiveTitleLineStyle(defaults.TitleLine, titleLineOverride)

	// log.Printf("DEBUG: Exiting getEffectiveCommentTextStyle. Effective MainOffset=%.2f, CrossOffset=%.2f", effective.MainAxisOffset, effective.CrossAxisOffset) // Log effective values
	return effective
}

func getEffectiveYearTextStyle(globalFont *FontStyle, defaults YearTextStyle, override *YearTextStyleOverride) YearTextStyle {
	effective := defaults
	fontOverride := (*FontStyleOverride)(nil) // Start with nil font override

	if override != nil {
		effective.Position = getString(override.Position, defaults.Position)
		effective.MainAxisOffset = getFloat64(override.MainAxisOffset, defaults.MainAxisOffset)
		effective.CrossAxisOffset = getFloat64(override.CrossAxisOffset, defaults.CrossAxisOffset)
		effective.TextColor = getString(override.TextColor, defaults.TextColor)
		effective.Shape = getString(override.Shape, defaults.Shape)
		effective.FillColor = getString(override.FillColor, defaults.FillColor)
		effective.BorderColor = getString(override.BorderColor, defaults.BorderColor)
		effective.BorderWidth = getFloat64(override.BorderWidth, defaults.BorderWidth)
		fontOverride = override.Font // Assign the font override struct if present
	}

	effective.Font = getEffectiveFontStyle(globalFont, defaults.Font, fontOverride)

	// Default border color to text color if not set? Or connector color? Let's leave empty for now.
	// if effective.BorderColor == "" && effective.Shape == "circle" { effective.BorderColor = effective.TextColor }

	return effective
}

func getEffectiveCenterlineProjectionStyle(defaults CenterlineProjectionStyle, override *CenterlineProjectionStyle) CenterlineProjectionStyle {
	if override == nil {
		return defaults
	}
	effective := defaults
	if override.Color != "" {
		effective.Color = override.Color
	}
	return effective
}

// --- SVG Dash Array Helper --- (No changes needed)
func getStrokeDashArray(styleType string, width int) string {
	// ... (implementation from previous step) ...
	dashArray := ""
	if width <= 0 {
		width = 1
	} // Ensure width is positive for calculations
	switch styleType {
	case "dotted":
		// Make dot size proportional to width, ensure space is larger than dot
		dashArray = fmt.Sprintf(` stroke-dasharray="%d %d"`, width, width*2)
	case "dashed":
		// Make dash size proportional to width
		dashArray = fmt.Sprintf(` stroke-dasharray="%d %d"`, width*4, width*2)
	}
	return dashArray
}

// --- XML/HTML Escaping --- (No changes needed)
func escapeXML(s string) string {
	// ... (implementation from previous step) ...
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&")
		case '<':
			buf.WriteString("<")
		case '>':
			buf.WriteString(">")
		case '"':
			buf.WriteString("\"")
		case '\'':
			buf.WriteString("'") // ' is not valid in HTML4
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

var escapeHTML = escapeXML

// --- Helper Function: Determine Cross-Axis Direction ---
// Returns -1 for "start" (top/left), +1 for "end" (bottom/right), considering alternation.
func getCrossAxisDirection(position string, index int, isHorizontal bool) float64 {
	isStart := false
	// Default to start if position is unknown or empty
	if position == "" {
		position = "start"
	}

	switch position {
	case "start":
		isStart = true
	case "end":
		isStart = false
	case "alternate-start-end":
		isStart = (index%2 == 0) // Even index = start
	case "alternate-end-start":
		isStart = (index%2 != 0) // Odd index = start
	default: // Default to start for safety
		isStart = true
		// log.Printf("Warning: Unknown position '%s', defaulting to 'start'", position)
	}

	if isStart {
		return -1.0 // Start = Top (-Y) or Left (-X)
	}
	return 1.0 // End = Bottom (+Y) or Right (+X)
}

// Helper to get effective FontStyle considering global, default, and override
func getEffectiveFontStyle(global *FontStyle, defaults FontStyle, override *FontStyleOverride) FontStyle {
	base := defaults // Start with the specific component's default

	// Apply global defaults if base values are zero/empty
	if global != nil {
		if base.FontFamily == "" {
			base.FontFamily = global.FontFamily
		}
		if base.FontSize == 0 {
			base.FontSize = global.FontSize
		}
		if base.FontWeight == "" {
			base.FontWeight = global.FontWeight
		}
		if base.FontStyle == "" {
			base.FontStyle = global.FontStyle
		}
	}

	// Apply hardcoded defaults if still zero/empty
	if base.FontFamily == "" {
		base.FontFamily = defaultFont
	} // Use constant
	if base.FontSize == 0 {
		base.FontSize = int(defaultFontSize)
	}
	if base.FontWeight == "" {
		base.FontWeight = "normal"
	}
	if base.FontStyle == "" {
		base.FontStyle = "normal"
	}

	effective := base // This now holds the fully defaulted base style

	// Apply override if provided
	if override != nil {
		effective.FontFamily = getString(override.FontFamily, base.FontFamily)
		effective.FontSize = getInt(override.FontSize, base.FontSize)
		effective.FontWeight = getString(override.FontWeight, base.FontWeight)
		effective.FontStyle = getString(override.FontStyle, base.FontStyle)
	}

	// Final fallback if Font Family is *still* empty - Restore sans-serif fallback
	if effective.FontFamily == "" {
		effective.FontFamily = "sans-serif"
	}

	return effective
}

// --- Text Dimension Estimation Helpers ---

// getEstimatedHeight provides a rough estimate of text height based on font size.
// SVG coordinates often need slight adjustments based on baseline, etc.
func getEstimatedHeight(font FontStyle) float64 {
	// Base height on font size, add a small buffer for typical line spacing/ascenders/descenders
	if font.FontSize <= 0 {
		return 15 // Default height if font size is invalid
	}
	return float64(font.FontSize) * 1.2
}

// estimateTextSVGWidth provides a very rough estimate of text width.
// Accurate SVG text width calculation is complex; this uses a simple heuristic.
func estimateTextSVGWidth(text string, font FontStyle) float64 {
	if font.FontSize <= 0 || text == "" {
		return 0
	}
	// Heuristic: average character width is roughly 0.6 * font size for proportional fonts
	averageCharWidthFactor := 0.6
	estimatedWidth := float64(len([]rune(text))) * float64(font.FontSize) * averageCharWidthFactor
	return estimatedWidth
}

// --- Shape String Parsing ---

// parsePadding parses a CSS-like padding string (e.g., "10", "10 20", "5 10 15 20")
// into individual top, right, bottom, left float values.
// Defaults to 0 if parsing fails or string is empty.
func parsePadding(paddingStr string) (float64, float64, float64, float64) {
	if paddingStr == "" {
		return 0, 0, 0, 0
	}

	parts := strings.Fields(paddingStr) // Split by whitespace
	values := make([]float64, 0, 4)

	for _, part := range parts {
		val, err := strconv.ParseFloat(part, 64)
		if err != nil {
			// log.Printf("Warning: Invalid padding value '%s', defaulting to 0: %v", part, err)
			values = append(values, 0) // Default invalid parts to 0
		} else {
			values = append(values, val)
		}
	}

	switch len(values) {
	case 1:
		return values[0], values[0], values[0], values[0] // top, right, bottom, left
	case 2:
		return values[0], values[1], values[0], values[1] // top/bottom, right/left
	case 3:
		return values[0], values[1], values[2], values[1] // top, right/left, bottom
	case 4:
		return values[0], values[1], values[2], values[3] // top, right, bottom, left
	default: // More than 4 or 0 after filtering errors
		if len(values) > 4 {
			return values[0], values[1], values[2], values[3] // Use first 4 if too many
		}
		return 0, 0, 0, 0 // Default if empty after errors
	}
}

// parseShapeString extracts shape type and parameters from a string like "circle;r=10".
// Returns shape type, a map of parameters, and an error if parsing fails.
func parseShapeString(shapeStr string) (string, map[string]float64, error) { // NOSONAR
	params := make(map[string]float64)
	if shapeStr == "" || shapeStr == "none" {
		return "none", params, nil
	}

	parts := strings.Split(shapeStr, ";")
	shapeType := strings.ToLower(strings.TrimSpace(parts[0]))

	if shapeType == "" {
		return "none", params, fmt.Errorf("shape string cannot start with ';'")
	}

	for _, part := range parts[1:] {
		paramParts := strings.SplitN(part, "=", 2)
		if len(paramParts) != 2 {
			return shapeType, params, fmt.Errorf("invalid parameter format in shape string: %s", part)
		}
		key := strings.ToLower(strings.TrimSpace(paramParts[0]))
		valStr := strings.TrimSpace(paramParts[1])

		// Handle special 'auto' value for radius
		if key == "r" && strings.ToLower(valStr) == "auto" {
			params[key] = -1 // Use -1 to signify 'auto' radius
		} else {
			val, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return shapeType, params, fmt.Errorf("invalid numeric value for parameter '%s': %s", key, valStr)
			}
			params[key] = val
		}
	}

	// Validate required parameters for known shapes
	switch shapeType {
	case "circle":
		if _, ok := params["r"]; !ok {
			return shapeType, params, fmt.Errorf("missing required parameter 'r' for circle shape")
		}
	case "rectangle":
		if _, ok := params["w"]; !ok {
			return shapeType, params, fmt.Errorf("missing required parameter 'w' for rectangle shape")
		}
		if _, ok := params["h"]; !ok {
			return shapeType, params, fmt.Errorf("missing required parameter 'h' for rectangle shape")
		}
	// Add validation for other shapes here if needed
	case "none":
		// No parameters needed
	default:
		// Allow unknown shapes but don't validate params
		// log.Printf("Warning: Unknown shape type '%s' encountered.", shapeType)
	}

	return shapeType, params, nil
}
