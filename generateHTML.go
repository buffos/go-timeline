// generateHTML.go
package main

import (
	"fmt"
	"log"
	"strings"
	// "math" // No longer needed here after CSS changes
)

// generateHTML creates a basic HTML representation of the timeline.
func generateHTML(template Template, entries []TimelineEntry) (string, error) { // NOSONAR
	var htmlBuilder strings.Builder

	// --- Basic HTML Structure ---
	htmlBuilder.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<title>Timeline</title>\n")
	htmlBuilder.WriteString("<style>\n")

	// --- Global Font Styles ---
	globalStyle := getEffectiveFontStyle(nil, *template.GlobalFont, nil)
	htmlBuilder.WriteString(fmt.Sprintf("body { margin: 0; padding: 40px; font-family: %s; font-size: %dpx; font-weight: %s; font-style: %s; }\n",
		escapeCSS(globalStyle.FontFamily), globalStyle.FontSize, escapeCSS(globalStyle.FontWeight), escapeCSS(globalStyle.FontStyle)))

	htmlBuilder.WriteString(".timeline-container { position: relative; margin: 20px auto; border: 1px solid #eee; /* Debug border */ }\n")

	// --- Center Line Style ---
	lineColor := template.CenterLine.Color
	if lineColor == "" {
		lineColor = "#000"
	}
	lineWidth := template.CenterLine.Width
	if lineWidth <= 0 {
		lineWidth = 2
	}
	lineStyle := "solid"
	if template.CenterLine.Type == "dotted" || template.CenterLine.Type == "dashed" {
		lineStyle = template.CenterLine.Type
	}
	isHorizontal := template.CenterLine.Orientation == "horizontal"

	// --- Estimate Container Size & Define Line ---
	containerHeight := 600.0   // Default height
	containerWidthCSS := "90%" // Default width (can be overridden below)
	// Calculate estimated total length along the main axis for container sizing
	totalAxisLength := template.Layout.Padding * 2 // Start with padding
	currentPosForLength := template.Layout.Padding
	for _, entry := range entries {
		spacing := template.Layout.EntrySpacing
		if entry.EntrySpacingOverride != nil {
			spacing = *entry.EntrySpacingOverride
		}
		if spacing <= 0 {
			spacing = template.Layout.EntrySpacing
		} // Fallback
		currentPosForLength += spacing
	}
	totalAxisLength = currentPosForLength // Total length is end position after last spacing

	if isHorizontal {
		containerHeight = 400                                      // Fixed height for horizontal example
		containerWidthCSS = fmt.Sprintf("%.0fpx", totalAxisLength) // Width based on content length + padding
		htmlBuilder.WriteString(fmt.Sprintf(
			`.center-line { position: absolute; left: %.0fpx; right: %.0fpx; top: 50%%; height: 0; border-top: %dpx %s %s; margin-top: -%dpx; }`,
			template.Layout.Padding, template.Layout.Padding, // Use padding for inset
			lineWidth, lineStyle, escapeCSS(lineColor), lineWidth/2,
		))
	} else { // Vertical
		containerHeight = totalAxisLength // Height based on content length + padding
		containerWidthCSS = "600px"       // Fixed width for vertical example (adjust as needed)
		htmlBuilder.WriteString(fmt.Sprintf(
			// Centerline positioned absolutely using percentages
			`.center-line { position: absolute; top: %.0fpx; bottom: %.0fpx; left: 50%%; width: 0; border-left: %dpx %s %s; margin-left: -%dpx; }`,
			template.Layout.Padding, template.Layout.Padding, // Use padding for inset
			lineWidth, lineStyle, escapeCSS(lineColor), lineWidth/2,
		))
	}
	// Apply calculated container dimensions
	htmlBuilder.WriteString(fmt.Sprintf(".timeline-container { height: %.0fpx; width: %s; }\n", containerHeight, containerWidthCSS))

	// --- Entry Styling ---
	// General style for absolutely positioned elements (year and comment)
	htmlBuilder.WriteString(`
        .timeline-element {
            position: absolute;
            z-index: 10;
            /* Base alignment - adjustments happen inline */
        }
	`)
	// Specific styles remain largely the same
	htmlBuilder.WriteString(`
        .year-text {
            text-align: center;
            white-space: nowrap;
        }
        .comment-box {
            font-size: 0.9em;
            padding: 8px;
            max-width: 180px;
            word-wrap: break-word;
            text-align: center;
            border-radius: 3px;
            background-color: #f8f8f8;
            border: 1px solid #ddd;
            position: relative; /* Needed if using pseudo-elements later */
            z-index: 5; /* Below year text if they overlap slightly */
        }
         .comment-box img { max-width: 100%; height: auto; display: block; margin: 5px auto; }
        a { color: inherit; text-decoration: none; }
        a:hover { text-decoration: underline; }
    `)
	htmlBuilder.WriteString("\n")
	htmlBuilder.WriteString("</style>\n</head>\n<body>\n")
	htmlBuilder.WriteString("<div class=\"timeline-container\">\n")
	htmlBuilder.WriteString("  <div class=\"center-line\"></div>\n")

	// --- Loop Through Entries ---
	currentPos := template.Layout.Padding // Start position from padding edge

	for i, entry := range entries {
		// --- Calculate Segment Details ---
		spacing := template.Layout.EntrySpacing
		if entry.EntrySpacingOverride != nil {
			spacing = *entry.EntrySpacingOverride
		}
		if spacing <= 0 {
			spacing = template.Layout.EntrySpacing
		} // Fallback
		entryCenterPos := currentPos + spacing/2.0 // Center point along the main axis

		// --- Determine Effective Styles ---
		yearStyle := getEffectiveYearTextStyle(template.GlobalFont, template.PeriodDefaults.YearText, entry.YearTextOverride)
		commentStyle := getEffectiveCommentTextStyle(template.GlobalFont, template.PeriodDefaults.CommentText, entry.CommentTextOverride)

		// --- Calculate Positioning Targets ---
		yearCrossAxisDir := getCrossAxisDirection(yearStyle.Position, i, isHorizontal)
		commentCrossAxisDir := getCrossAxisDirection(commentStyle.Position, i, isHorizontal)

		baseConnectorLength := template.Layout.ConnectorLength
		yearTargetX, yearTargetY := 0.0, 0.0       // Target coords for year element's anchor
		commentTargetX, commentTargetY := 0.0, 0.0 // Target coords for comment element's anchor

		if isHorizontal {
			// Year position
			yearTargetX = entryCenterPos // Remove + yearStyle.MainAxisOffset
			yearTargetY = (containerHeight / 2.0) + (yearCrossAxisDir * (baseConnectorLength /* + yearStyle.CrossAxisOffset - Use yearStyle.Offset here if needed */))

			// Comment position
			commentTargetX = entryCenterPos // Remove + commentStyle.MainAxisOffset
			commentTargetY = (containerHeight / 2.0) + (commentCrossAxisDir * (baseConnectorLength /* + commentStyle.CrossAxisOffset - Comment doesn't have simple offset */))
		} else { // Vertical
			// Year position - Use percentage for X-axis (left: 50%) and adjust with transform
			yearTargetY = entryCenterPos // Remove + yearStyle.MainAxisOffset // Y position along the axis
			// X target represents the offset from the center line
			yearTargetX = yearCrossAxisDir * (baseConnectorLength /* + yearStyle.CrossAxisOffset - Use yearStyle.Offset here if needed */)

			// Comment position - Use percentage for X-axis
			commentTargetY = entryCenterPos // Remove + commentStyle.MainAxisOffset // Y position along the axis
			// X target represents the offset from the center line
			commentTargetX = commentCrossAxisDir * (baseConnectorLength /* + commentStyle.CrossAxisOffset - Comment doesn't have simple offset */)
		}

		// --- Year Text Element ---
		yearFont := yearStyle.Font
		yearColor := yearStyle.TextColor
		if yearColor == "" {
			yearColor = "inherit"
		}

		yearInlineStyle := fmt.Sprintf("color:%s; font-family:%s; font-size:%dpx; font-weight:%s; font-style:%s;",
			escapeCSS(yearColor), escapeCSS(yearFont.FontFamily), yearFont.FontSize, escapeCSS(yearFont.FontWeight), escapeCSS(yearFont.FontStyle))

		// CSS positioning styles
		yearPosStyle := ""
		if isHorizontal {
			// Position top-left corner, then use transform to center/align
			yearPosStyle = fmt.Sprintf("left: %.0fpx; top: %.0fpx; transform: translate(-50%%, %s);",
				yearTargetX, yearTargetY,
				ternary(yearCrossAxisDir < 0, "-100%", "0%")) // Shift up if above line
		} else { // Vertical
			// Position top relative to axis, use left: 50% and transform for horizontal offset
			yearPosStyle = fmt.Sprintf("top: %.0fpx; left: 50%%; transform: translate(%s, -50%%);", // Center vertically
				yearTargetY,
				ternary(yearCrossAxisDir < 0, fmt.Sprintf("calc(-100%% + %.0fpx)", yearTargetX), fmt.Sprintf("%.0fpx", yearTargetX))) // Apply offset from center
			// Adjust text alignment based on side
			if yearCrossAxisDir < 0 {
				yearInlineStyle += " text-align: right;"
			} else {
				yearInlineStyle += " text-align: left;"
			}
		}

		linkOpenTag := ""
		linkCloseTag := ""
		if entry.Link != "" {
			linkOpenTag = fmt.Sprintf(`<a href="%s" target="_blank">`, escapeHTML(entry.Link))
			linkCloseTag = `</a>`
		}
		htmlBuilder.WriteString(fmt.Sprintf("  <div class=\"timeline-element year-text-container\" style=\"%s\">\n", yearPosStyle)) // Apply positioning
		htmlBuilder.WriteString(fmt.Sprintf("    %s<div class=\"year-text\" style=\"%s\">%s</div>%s\n", linkOpenTag, yearInlineStyle, escapeHTML(entry.Period), linkCloseTag))
		htmlBuilder.WriteString("  </div>\n") // Close year-text-container

		// --- Comment Element (if exists) ---
		if entry.CommentText != "" || entry.CommentImage != "" {
			commentFont := commentStyle.Font
			commentTextColor := commentStyle.TextColor
			if commentTextColor == "" {
				commentTextColor = "inherit"
			}

			// Base style for the comment box div content
			commentBoxStyle := fmt.Sprintf("color:%s; font-family:%s; font-size:%dpx; font-weight:%s; font-style:%s;",
				escapeCSS(commentTextColor), escapeCSS(commentFont.FontFamily), commentFont.FontSize, escapeCSS(commentFont.FontWeight), escapeCSS(commentFont.FontStyle))

			// Add shape styling
			if commentStyle.Shape == "rectangle" {
				bgColor := commentStyle.FillColor
				if bgColor != "" {
					commentBoxStyle += fmt.Sprintf(" background-color:%s;", escapeCSS(bgColor))
				} else {
					commentBoxStyle += " background-color: transparent;"
				}
				borderColor := commentStyle.BorderColor
				borderWidth := commentStyle.BorderWidth
				if borderColor != "" && borderWidth > 0 {
					borderStyle := commentStyle.BorderStyle
					if borderStyle == "" {
						borderStyle = "solid"
					}
					commentBoxStyle += fmt.Sprintf(" border: %dpx %s %s;", borderWidth, escapeCSS(borderStyle), escapeCSS(borderColor))
				} else {
					commentBoxStyle += " border: none;"
				}
				// Parse padding string and apply
				padTop, padRight, padBottom, padLeft := parsePadding(commentStyle.Padding)
				commentBoxStyle += fmt.Sprintf(" padding: %.0fpx %.0fpx %.0fpx %.0fpx;", padTop, padRight, padBottom, padLeft)
			} else { // shape == "none"
				commentBoxStyle += " background-color: transparent; border: none; padding: 0;"
			}

			// CSS positioning styles for the comment container
			commentPosStyle := ""
			if isHorizontal {
				commentPosStyle = fmt.Sprintf("left: %.0fpx; top: %.0fpx; transform: translate(-50%%, %s);",
					commentTargetX, commentTargetY,
					ternary(commentCrossAxisDir < 0, "-100%", "0%")) // Shift up if above line
			} else { // Vertical
				commentPosStyle = fmt.Sprintf("top: %.0fpx; left: 50%%; transform: translate(%s, -50%%);", // Center vertically
					commentTargetY,
					ternary(commentCrossAxisDir < 0, fmt.Sprintf("calc(-100%% + %.0fpx)", commentTargetX), fmt.Sprintf("%.0fpx", commentTargetX))) // Offset from center
				// Adjust text alignment based on side
				if commentCrossAxisDir < 0 {
					commentBoxStyle += " text-align: right;" // Align text inside box
				} else {
					commentBoxStyle += " text-align: left;"
				}
			}

			imageTag := ""
			if entry.CommentImage != "" {
				imageTag = fmt.Sprintf(`<img src="%s" alt="Timeline image"/>`, escapeHTML(entry.CommentImage))
			}
			commentContent := entry.CommentText // Allow HTML

			htmlBuilder.WriteString(fmt.Sprintf("  <div class=\"timeline-element comment-box-container\" style=\"%s\">\n", commentPosStyle)) // Apply positioning
			htmlBuilder.WriteString(fmt.Sprintf("    <div class=\"comment-box\" style=\"%s\">%s%s</div>\n", commentBoxStyle, imageTag, commentContent))
			htmlBuilder.WriteString("  </div>\n") // Close comment-box-container
		}

		// --- Advance position for the next entry ---
		currentPos += spacing
	}

	htmlBuilder.WriteString("</div>\n") // Close timeline-container
	htmlBuilder.WriteString("</body>\n</html>")

	log.Println("Warning: HTML output is simplified. Connectors, dots, and precise layout/overlap avoidance are not fully implemented.")
	return htmlBuilder.String(), nil
}

// Simple CSS Escaping (basic)
func escapeCSS(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// Simple ternary helper for inline conditions
func ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
