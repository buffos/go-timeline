package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Constants (Consider moving some to LayoutOptions in Template)
const defaultFontSize = 12.0
const defaultFont = "Arial, sans-serif"
const imagePlaceholderHeight = 50.0       // Default height for images if not specified/calculable
const foreignObjectHeightEstimate = 100.0 // Default height for foreignObject (adjust as needed) - VERY ROUGH

// Structure to hold calculated bounds
type bounds struct {
	minX, maxX, minY, maxY float64
	isSet                  bool
}

// Update bounds considering a point (x, y)
func (b *bounds) updatePoint(x, y float64) {
	if !b.isSet {
		b.minX, b.maxX = x, x
		b.minY, b.maxY = y, y
		b.isSet = true
	} else {
		b.minX = math.Min(b.minX, x)
		b.maxX = math.Max(b.maxX, x)
		b.minY = math.Min(b.minY, y)
		b.maxY = math.Max(b.maxY, y)
	}
}

// Update bounds considering a rectangle
func (b *bounds) updateRect(x, y, width, height float64) {
	if width > 0 && height > 0 {
		b.updatePoint(x, y)
		b.updatePoint(x+width, y+height)
	}
}

// Parameter structs for functions with too many parameters
type ElementCenterParams struct {
	AxisX        float64
	AxisY        float64
	MainOffset   float64
	CrossOffset  float64
	ConnectorLen float64
	CrossDir     float64
	IsHorizontal bool
}

type JunctionMarkerParams struct {
	Style           JunctionMarkerStyle
	CenterX         float64
	CenterY         float64
	MarkerColor     string
	IsHorizontal    bool
	CenterLineWidth float64
}

type CommentParams struct {
	Style        CommentTextStyle
	AnchorX      float64
	AnchorY      float64
	CrossAxisDir float64
	IsHorizontal bool
	SegmentWidth float64
	DefaultColor string
	TitleText    string
	BodyText     string
	ImageURL     string
}

// Add a new parameter struct for drawConnector
type ConnectorParams struct {
	X1                 float64
	Y1                 float64
	X2                 float64
	Y2                 float64
	Style              ConnectorStyle
	SegmentColor       string
	IsHorizontal       bool
	CrossAxisDir       float64
	LineIsVisible      bool
	ElementCrossOffset float64 // Offset of the connected element (year/comment)
}

// Add a new parameter struct for drawYearShape
type YearShapeParams struct {
	ShapeType   string
	ShapeParams map[string]float64
	CenterX     float64
	CenterY     float64
	TextWidth   float64
	TextHeight  float64
	YearStyle   YearTextStyle
}

// Add a parameter struct for drawConnectorDot
type ConnectorDotParams struct {
	DotStyle      DotStyle
	P1x, P1y      float64
	P2x, P2y      float64
	DefaultColor  string
	IsHorizontal  bool
	CrossAxisDir  float64
	LineIsVisible bool
}

// Add a parameter struct for drawCenterLineSegment
type DrawCenterLineSegmentParams struct {
	SVG         *bytes.Buffer
	Bounds      *bounds
	X1, Y1      float64
	X2, Y2      float64
	Color       string
	Width       float64
	LineType    string
	RoundedCaps bool
}

// Add a parameter struct for drawAndAdvanceAxisSegment
type DrawAndAdvanceAxisSegmentParams struct {
	SVG                *bytes.Buffer
	Bounds             *bounds
	CurrentX, CurrentY float64
	StartIndex         int // -1 for initial segment, 0 to len(entries)-2 for others
	Entries            []TimelineEntry
	Data               TimelinePositionData
	LayoutConfig       LayoutConfig
	BaseOrientation    string
	GlobalAxisAngle    *float64
	CenterLineType     string
}

// calculateAxisGeometry determines the start/end coordinates and effective angle
// for a segment of the timeline axis based on orientation and angle overrides.
func calculateAxisGeometry(x1, y1, length float64, orientation string, globalAngle, overrideAngle *float64) (nx1, ny1, nx2, ny2, effectiveAngleDeg float64) {
	// Determine base angle from orientation
	var baseAngleDeg float64
	if orientation == "vertical" {
		baseAngleDeg = 90.0 // Upwards
	} else {
		baseAngleDeg = 0.0 // Rightwards (default horizontal)
	}

	// Determine the effective angle in degrees
	effectiveAngleDeg = baseAngleDeg // Start with the orientation angle
	if globalAngle != nil {
		effectiveAngleDeg = *globalAngle // Override with global angle if set
	}
	if overrideAngle != nil {
		effectiveAngleDeg = *overrideAngle // Override with entry-specific angle if set
	}

	// Convert effective angle to radians for trig functions
	effectiveAngleRad := effectiveAngleDeg * math.Pi / 180.0

	// Calculate end point coordinates
	nx1 = x1
	ny1 = y1
	nx2 = x1 + length*math.Cos(effectiveAngleRad)
	ny2 = y1 + length*math.Sin(effectiveAngleRad)

	return nx1, ny1, nx2, ny2, effectiveAngleDeg
}

// --- Helper Functions for Timeline Generation ---

// LayoutConfig holds the configuration for timeline layout
type LayoutConfig struct {
	layoutPadding          float64
	defaultEntrySpacing    float64
	defaultConnectorLength float64
	centerLineBaseColor    string
	centerLineWidth        float64
	centerLineIsRounded    bool
}

// TimelinePositionData holds pre-calculated data for timeline entries
type TimelinePositionData struct {
	entryPoints     []float64
	junctionPoints  []float64
	segmentColors   []string
	markerStyles    []JunctionMarkerStyle
	connectorStyles []ConnectorStyle
	yearStyles      []YearTextStyle
	commentStyles   []CommentTextStyle
}

// CommentBlockLayout holds layout information for comment blocks
type CommentBlockLayout struct {
	blockX, blockY     float64 // Top-left corner of the visual block
	visualBlockWidth   float64 // Width of the visual block (content + L/R padding)
	visualBlockHeight  float64 // Height of the visual block (content + T/B padding)
	contentCenterX     float64 // Center X relative to content area
	titleTextAbsY      float64
	titleLineAbsY      float64
	bodyAbsX, bodyAbsY float64 // Top-left corner of the foreignObject
	foHeight           float64 // Estimated height of content *within* foreignObject
	// Parsed padding values
	padTop, padRight, padBottom, padLeft float64
	contentWidth                         float64 // Width available for content inside padding (FO width)
}

// Initialize layout configuration from template
func initializeLayoutConfig(template Template) LayoutConfig {
	config := LayoutConfig{}

	config.layoutPadding = template.Layout.Padding
	if config.layoutPadding <= 0 {
		config.layoutPadding = 50.0
	}

	config.defaultEntrySpacing = template.Layout.EntrySpacing
	if config.defaultEntrySpacing <= 0 {
		config.defaultEntrySpacing = 150.0
	}

	config.defaultConnectorLength = template.Layout.ConnectorLength
	if config.defaultConnectorLength <= 0 {
		config.defaultConnectorLength = 50.0
	}

	config.centerLineBaseColor = template.CenterLine.Color
	if config.centerLineBaseColor == "" {
		config.centerLineBaseColor = "#000000"
	}

	config.centerLineWidth = float64(template.CenterLine.Width)
	if config.centerLineWidth <= 0 {
		config.centerLineWidth = 2
	}

	config.centerLineIsRounded = template.CenterLine.RoundedCaps

	return config
}

// Calculate timeline positions and styles
func calculateTimelinePositionsAndStyles(entries []TimelineEntry, template Template, config LayoutConfig) TimelinePositionData {
	data := TimelinePositionData{
		entryPoints:     make([]float64, len(entries)),
		junctionPoints:  make([]float64, len(entries)+1),
		segmentColors:   make([]string, len(entries)),
		markerStyles:    make([]JunctionMarkerStyle, len(entries)),
		connectorStyles: make([]ConnectorStyle, len(entries)),
		yearStyles:      make([]YearTextStyle, len(entries)),
		commentStyles:   make([]CommentTextStyle, len(entries)),
	}

	currentPos := 0.0

	for i, entry := range entries {
		// Spacing
		spacing := config.defaultEntrySpacing
		if entry.EntrySpacingOverride != nil {
			spacing = *entry.EntrySpacingOverride
		}
		if spacing <= 0 {
			spacing = config.defaultEntrySpacing
		}

		// Positions
		data.junctionPoints[i] = currentPos
		data.entryPoints[i] = currentPos + spacing/2.0
		currentPos += spacing

		// Styles
		projStyle := getEffectiveCenterlineProjectionStyle(template.PeriodDefaults.CenterlineProjection, entry.CenterlineProjectionOverride)
		data.segmentColors[i] = projStyle.Color
		if data.segmentColors[i] == "" {
			data.segmentColors[i] = config.centerLineBaseColor
		}

		data.markerStyles[i] = getEffectiveJunctionMarkerStyle(template.PeriodDefaults.JunctionMarker, entry.JunctionMarkerOverride)
		data.connectorStyles[i] = getEffectiveConnectorStyle(template.PeriodDefaults.Connector, entry.ConnectorOverride)
		data.yearStyles[i] = getEffectiveYearTextStyle(template.GlobalFont, template.PeriodDefaults.YearText, entry.YearTextOverride)
		data.commentStyles[i] = getEffectiveCommentTextStyle(template.GlobalFont, template.PeriodDefaults.CommentText, entry.CommentTextOverride)
	}
	data.junctionPoints[len(entries)] = currentPos

	return data
}

// Add a parameter struct for drawTimelineEntry
type TimelineEntryParams struct {
	Index        int
	Entry        TimelineEntry
	Data         TimelinePositionData
	EntryAxisX   float64 // X coordinate of the entry on the potentially angled axis
	EntryAxisY   float64 // Y coordinate of the entry on the potentially angled axis
	IsHorizontal bool    // True if base orientation is horizontal (for annotation direction)
	Config       LayoutConfig
}

// Update the drawTimelineEntry function to handle connectors correctly based on config
func drawTimelineEntry(svg *bytes.Buffer, bounds *bounds, params TimelineEntryParams) {
	i := params.Index
	entry := params.Entry
	timelineData := params.Data
	entryAxisX := params.EntryAxisX // Use the passed exact coordinates
	entryAxisY := params.EntryAxisY // Use the passed exact coordinates
	// Determine effective orientation for *this specific entry*
	effectiveIsHorizontal := params.IsHorizontal // Start with global orientation
	if entry.OrientationOverride != nil {
		if *entry.OrientationOverride == "horizontal" {
			effectiveIsHorizontal = true
		} else if *entry.OrientationOverride == "vertical" {
			effectiveIsHorizontal = false
		}
		// Ignore invalid override values, keep global default
	}
	config := params.Config

	// --- Get Styles for this entry ---
	connStyle := timelineData.connectorStyles[i]
	commentStyle := timelineData.commentStyles[i]
	yearStyle := timelineData.yearStyles[i]
	markerStyle := timelineData.markerStyles[i]
	segmentColor := timelineData.segmentColors[i] // Color of segment LEADING to this entry

	// Determine cross-axis direction based on *effective* orientation
	commentCrossAxisDir := 1.0
	yearCrossAxisDir := -1.0
	if i%2 != 0 { // Alternate sides
		commentCrossAxisDir = -1.0
		yearCrossAxisDir = 1.0
	}
	// Allow override for connector side, checking against *effective* orientation
	if connStyle.Side != "" {
		if (effectiveIsHorizontal && connStyle.Side == "top") || (!effectiveIsHorizontal && connStyle.Side == "left") {
			commentCrossAxisDir = -1.0
			yearCrossAxisDir = 1.0 // Year goes opposite comment
		} else if (effectiveIsHorizontal && connStyle.Side == "bottom") || (!effectiveIsHorizontal && connStyle.Side == "right") {
			commentCrossAxisDir = 1.0
			yearCrossAxisDir = -1.0 // Year goes opposite comment
		}
	}

	// --- Junction Marker ---
	markerColor := determineMarkerColor(markerStyle, segmentColor, connStyle)
	drawJunctionMarker(svg, bounds, JunctionMarkerParams{
		Style:           markerStyle,
		CenterX:         entryAxisX,
		CenterY:         entryAxisY,
		MarkerColor:     markerColor,
		IsHorizontal:    effectiveIsHorizontal, // Use effective orientation
		CenterLineWidth: config.centerLineWidth,
	})

	// --- Year Element ---
	// Calculate center based on axis point and *effective* orientation
	yearCenterX, yearCenterY := calculateElementCenter(ElementCenterParams{
		AxisX:        entryAxisX,
		AxisY:        entryAxisY,
		MainOffset:   yearStyle.MainAxisOffset,
		CrossOffset:  yearStyle.CrossAxisOffset,
		ConnectorLen: config.defaultConnectorLength,
		CrossDir:     yearCrossAxisDir,
		IsHorizontal: effectiveIsHorizontal,
	})

	// --- Draw Connector to Year Element (Restored Logic) ---
	drawPeriodLine := connStyle.DrawToPeriod == nil || *connStyle.DrawToPeriod
	if drawPeriodLine {
		drawConnector(svg, bounds, ConnectorParams{
			X1:                 yearCenterX,
			Y1:                 yearCenterY,
			X2:                 entryAxisX,
			Y2:                 entryAxisY,
			Style:              connStyle,
			SegmentColor:       segmentColor,
			IsHorizontal:       effectiveIsHorizontal,
			CrossAxisDir:       yearCrossAxisDir,
			LineIsVisible:      drawPeriodLine,
			ElementCrossOffset: yearStyle.CrossAxisOffset,
		})
	}

	// --- Draw Year Element itself ---
	drawYearElement(svg, bounds, entry, yearStyle, yearCenterX, yearCenterY)

	// --- Comment Element and Connector ---
	if entry.CommentText != "" || entry.TitleText != "" || entry.CommentImage != "" {
		// Calculate Comment Anchor Point using *effective* orientation
		commentAnchorX, commentAnchorY := calculateElementCenter(ElementCenterParams{
			AxisX:        entryAxisX,
			AxisY:        entryAxisY,
			MainOffset:   commentStyle.MainAxisOffset,
			CrossOffset:  commentStyle.CrossAxisOffset,
			ConnectorLen: config.defaultConnectorLength,
			CrossDir:     commentCrossAxisDir,
			IsHorizontal: effectiveIsHorizontal,
		})

		// Calculate comment block layout based on the anchor point and *effective* orientation
		blockLayout := calculateCommentBlockLayout(CommentParams{
			Style:        commentStyle,
			AnchorX:      commentAnchorX,
			AnchorY:      commentAnchorY,
			CrossAxisDir: commentCrossAxisDir,
			IsHorizontal: effectiveIsHorizontal,
			SegmentWidth: config.defaultEntrySpacing,
			DefaultColor: connStyle.Color,
			TitleText:    entry.TitleText,
			BodyText:     entry.CommentText,
			ImageURL:     entry.CommentImage,
		})

		// Determine comment edge point based on *effective* orientation
		commentEdgeX, commentEdgeY := calculateCommentEdgePoint(blockLayout, commentCrossAxisDir, effectiveIsHorizontal)

		// --- Draw Connector to comment using *effective* orientation
		drawCommentLine := connStyle.DrawToComment == nil || *connStyle.DrawToComment
		drawConnector(svg, bounds, ConnectorParams{
			X1:                 commentEdgeX,
			Y1:                 commentEdgeY,
			X2:                 entryAxisX,
			Y2:                 entryAxisY,
			Style:              connStyle,
			SegmentColor:       segmentColor,
			IsHorizontal:       effectiveIsHorizontal,
			CrossAxisDir:       commentCrossAxisDir,
			LineIsVisible:      drawCommentLine,
			ElementCrossOffset: commentStyle.CrossAxisOffset,
		})

		// --- Draw Comment Block ---
		drawComment(svg, bounds, CommentParams{
			Style:        commentStyle,
			AnchorX:      commentAnchorX,
			AnchorY:      commentAnchorY,
			CrossAxisDir: commentCrossAxisDir,
			IsHorizontal: effectiveIsHorizontal,
			SegmentWidth: config.defaultEntrySpacing,
			DefaultColor: connStyle.Color,
			TitleText:    entry.TitleText,
			BodyText:     entry.CommentText,
			ImageURL:     entry.CommentImage,
		})
	}
}

// --- Helper to find the edge point of the comment box ---
func calculateCommentEdgePoint(layout CommentBlockLayout, crossAxisDir float64, isHorizontal bool) (float64, float64) {
	// Calculate the center of the edge facing the timeline axis
	if isHorizontal {
		if crossAxisDir < 0 { // Top edge center
			return layout.blockX + layout.visualBlockWidth/2.0, layout.blockY
		} else { // Bottom edge center
			return layout.blockX + layout.visualBlockWidth/2.0, layout.blockY + layout.visualBlockHeight
		}
	} else { // Vertical
		if crossAxisDir < 0 { // Left edge center
			return layout.blockX, layout.blockY + layout.visualBlockHeight/2.0
		} else { // Right edge center
			return layout.blockX + layout.visualBlockWidth, layout.blockY + layout.visualBlockHeight/2.0
		}
	}
}

// Determine the color for a marker based on style and defaults
func determineMarkerColor(markerStyle JunctionMarkerStyle, segmentColor string, connStyle ConnectorStyle) string {
	markerColor := segmentColor   // Marker color matches current segment/connector color
	if markerStyle.Color != nil { // Allow explicit override
		markerColor = *markerStyle.Color
	} else {
		// Fallback further to connector color if segment color is bland?
		if connStyle.Color != "" {
			markerColor = connStyle.Color
		}
	}
	return markerColor
}

// --- Helper function to determine connector line style attributes ---
func calculateConnectorStyleAttributes(style ConnectorStyle, segmentColor string) (string, float64, string) {
	connDrawColor := style.Color
	if connDrawColor == "" {
		connDrawColor = segmentColor
	}
	connDrawWidth := float64(style.Width)
	if connDrawWidth <= 0 {
		connDrawWidth = 1
	}
	connDashArray := getStrokeDashArray(style.LineType, int(connDrawWidth))
	return connDrawColor, connDrawWidth, connDashArray
}

// --- Helper function to calculate direction vectors for the connector ---
func calculateConnectorVectors(x1, y1, x2, y2 float64) (ux, uy, nx, ny, lineLen float64) {
	// Vectors point FROM axis (X2,Y2) TOWARDS comment edge (X1,Y1)
	dx := x1 - x2
	dy := y1 - y2
	lineLen = math.Sqrt(dx*dx + dy*dy)
	ux, uy = 1.0, 0.0 // Default unit vector along line from X2 towards X1
	nx, ny = 0.0, 1.0 // Default unit vector perpendicular
	if lineLen > 0.001 {
		ux = dx / lineLen
		uy = dy / lineLen
		nx = -uy // Perpendicular: Points left relative to direction X2->X1
		ny = ux
	}
	return ux, uy, nx, ny, lineLen
}

// --- Helper function to calculate the absolute dot position ---
func calculateConnectorDotPosition(axisX, axisY, ux, uy, nx, ny float64, dotStyle DotStyle) (dotX, dotY float64) {
	offsetMain := float64(dotStyle.OffsetMain)
	offsetCross := float64(dotStyle.OffsetCross)
	// Dot position = Axis Point + Main Offset along X2->X1 + Cross Offset perpendicular to X2->X1
	dotX = axisX + ux*offsetMain + nx*offsetCross
	dotY = axisY + uy*offsetMain + ny*offsetCross
	return dotX, dotY
}

// --- Parameter struct for drawConnectorLineSegments ---
type ConnectorLineSegmentsParams struct {
	SVG            *bytes.Buffer
	Bounds         *bounds
	ConnParams     ConnectorParams // Original ConnectorParams for context
	DotX, DotY     float64         // Calculated dot position
	Ux, Uy, Nx, Ny float64         // Direction vectors (Ux, Uy is Axis->Element; Nx, Ny is perpendicular)
	DrawWidth      float64
	DrawColor      string
	DashArray      string
}

// --- Helper function to draw the connector line segments ---
func drawConnectorLineSegments(params ConnectorLineSegmentsParams) {
	// Only draw if the line is marked as visible in the original connector params
	if !params.ConnParams.LineIsVisible {
		return
	}

	dotStyle := params.ConnParams.Style.Dot

	if !dotStyle.StopAtDot {
		// Case 1: Line does NOT stop at dot - Draw straight line from element (X1,Y1) to axis point (X2,Y2)
		fmt.Fprintf(params.SVG, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s />`,
			params.ConnParams.X1, params.ConnParams.Y1, params.ConnParams.X2, params.ConnParams.Y2,
			params.DrawColor, params.DrawWidth, params.DashArray)
		params.SVG.WriteString("\n")
		params.Bounds.updatePoint(params.ConnParams.X1, params.ConnParams.Y1)
		params.Bounds.updatePoint(params.ConnParams.X2, params.ConnParams.Y2)

	} else { // Case 2: Line STOPS at dot
		// Determine if a dogleg is needed based on dot OR element offset
		dotOffsetCross := float64(dotStyle.OffsetCross)
		elementOffsetCross := params.ConnParams.ElementCrossOffset

		isDogleg := math.Abs(dotOffsetCross) >= 0.001 || math.Abs(elementOffsetCross) >= 0.001

		if isDogleg {
			// Subcase 2a: Dogleg line Element(X1,Y1) -> midPoint -> Dot(DotX, DotY)

			// Calculate midpoint (elbow) based on orientation:
			var midPointX, midPointY float64
			if params.ConnParams.IsHorizontal {
				// Horizontal: Midpoint aligns vertically with Dot, horizontally with Element
				midPointX = params.ConnParams.X1 // Same X as element
				midPointY = params.DotY          // Same Y as dot
			} else {
				// Vertical: Midpoint aligns horizontally with Dot, vertically with Element
				midPointX = params.DotX          // Same X as dot
				midPointY = params.ConnParams.Y1 // Same Y as element
			}

			// Draw segment 1: Element (X1, Y1) to Midpoint
			fmt.Fprintf(params.SVG, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s />`,
				params.ConnParams.X1, params.ConnParams.Y1, midPointX, midPointY,
				params.DrawColor, params.DrawWidth, params.DashArray)
			params.SVG.WriteString("\n")
			// Draw segment 2: Midpoint to Dot
			fmt.Fprintf(params.SVG, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s />`,
				midPointX, midPointY, params.DotX, params.DotY,
				params.DrawColor, params.DrawWidth, params.DashArray)
			params.SVG.WriteString("\n")

			params.Bounds.updatePoint(params.ConnParams.X1, params.ConnParams.Y1)
			params.Bounds.updatePoint(midPointX, midPointY)
			params.Bounds.updatePoint(params.DotX, params.DotY)

		} else {
			// Subcase 2b: No dogleg, draw single line Element(X1, Y1) -> Dot(DotX, DotY)
			// (Handle potential zero-length line by drawing to axis point if needed)
			finalEndX, finalEndY := params.DotX, params.DotY

			// Check if stopping at the dot would create a zero-length line between Element and Dot
			isZeroLengthStop := math.Abs(params.DotX-params.ConnParams.X1) < 0.001 && math.Abs(params.DotY-params.ConnParams.Y1) < 0.001
			if isZeroLengthStop {
				// Avoid zero-length line: Draw to the original axis point instead
				finalEndX, finalEndY = params.ConnParams.X2, params.ConnParams.Y2
			}

			// Draw the single line segment
			fmt.Fprintf(params.SVG, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s />`,
				params.ConnParams.X1, params.ConnParams.Y1, finalEndX, finalEndY,
				params.DrawColor, params.DrawWidth, params.DashArray)
			params.SVG.WriteString("\n")
			params.Bounds.updatePoint(params.ConnParams.X1, params.ConnParams.Y1)
			params.Bounds.updatePoint(finalEndX, finalEndY)
		}
	}
}

// --- Refactored drawConnector function ---
// Orchestrates drawing the connector by calling helper functions.
func drawConnector(svg *bytes.Buffer, bounds *bounds, params ConnectorParams) {
	// 1. Calculate Style Attributes
	connDrawColor, connDrawWidth, connDashArray := calculateConnectorStyleAttributes(params.Style, params.SegmentColor)

	// 2. Calculate Direction Vectors (from axis X2,Y2 towards comment X1,Y1)
	ux, uy, nx, ny, _ := calculateConnectorVectors(params.X1, params.Y1, params.X2, params.Y2)

	// 3. Calculate Dot Position (relative to axis X2,Y2)
	dotStyle := params.Style.Dot
	dotX, dotY := calculateConnectorDotPosition(params.X2, params.Y2, ux, uy, nx, ny, dotStyle)

	// 4. Draw Line Segment(s) if visible
	drawConnectorLineSegments(ConnectorLineSegmentsParams{
		SVG:        svg,
		Bounds:     bounds,
		ConnParams: params, // Pass original params for context
		DotX:       dotX,
		DotY:       dotY,
		Ux:         ux,
		Uy:         uy,
		Nx:         nx,
		Ny:         ny,
		DrawWidth:  connDrawWidth,
		DrawColor:  connDrawColor,
		DashArray:  connDashArray,
	})

	// 5. Draw the Dot itself (if visible)
	drawConnectorDot(svg, bounds, ConnectorDotParams{
		DotStyle:      dotStyle,
		P1x:           params.X1, // Comment edge point
		P1y:           params.Y1,
		P2x:           params.X2, // Axis point
		P2y:           params.Y2,
		DefaultColor:  connDrawColor,
		IsHorizontal:  params.IsHorizontal,
		CrossAxisDir:  params.CrossAxisDir,
		LineIsVisible: params.LineIsVisible,
	}, dotX, dotY) // Pass calculated dot position
}

// --- Helper function to draw the connector dot ---
// Update parameters: Pass calculated dot center (dotX, dotY) and reference points for arrow logic (params includes P1x/y, P2x/y)
func drawConnectorDot(svg *bytes.Buffer, bounds *bounds, params ConnectorDotParams, dotX, dotY float64) {
	// Check if the dot style itself is visible/valid (Line visibility checked before calling drawConnector)
	if !params.DotStyle.Visible || params.DotStyle.Shape == "none" || params.DotStyle.Size <= 0 {
		return
	}

	dotSize := float64(params.DotStyle.Size)
	halfDotSize := dotSize / 2.0
	dotColor := params.DotStyle.Color
	if dotColor == "" {
		dotColor = params.DefaultColor // Default to connector color
	}

	// Dot position (dotX, dotY) is pre-calculated

	// Arrow logic uses reference points from params (P1x/y, P2x/y)
	// Remove redundant calculations - only needed for arrow orientation logic below
	// dx := params.P2x - params.P1x
	// dy := params.P2y - params.P1y
	// lineLen := math.Sqrt(dx*dx + dy*dy)
	// nx, ny := 0.0, 1.0 // Default perpendicular vector
	// if lineLen > 0.001 {
	// 	// Normalized perpendicular vector: (-dy/lineLen, dx/lineLen)
	// 	nx = -dy / lineLen
	// 	ny = dx / lineLen
	// }

	switch params.DotStyle.Shape {
	case "circle":
		fmt.Fprintf(svg, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>\n`,
			dotX, dotY, halfDotSize, dotColor)
	case "square":
		rectX := dotX - halfDotSize
		rectY := dotY - halfDotSize
		fmt.Fprintf(svg, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>\n`,
			rectX, rectY, dotSize, dotSize, dotColor)
	case "arrow":
		var p1xArrow, p1yArrow, p2xArrow, p2yArrow, tipX, tipY float64
		// Arrow points towards the axis (determined by CrossAxisDir)
		if params.IsHorizontal {
			// Base is horizontal, relative to dotX, dotY
			p1xArrow, p1yArrow = dotX-halfDotSize, dotY
			p2xArrow, p2yArrow = dotX+halfDotSize, dotY
			// Tip points vertically towards axis
			tipX = dotX
			tipY = dotY - params.CrossAxisDir*halfDotSize*1.2
		} else { // Vertical timeline
			// Base is vertical, relative to dotX, dotY
			p1xArrow, p1yArrow = dotX, dotY-halfDotSize
			p2xArrow, p2yArrow = dotX, dotY+halfDotSize
			// Tip points horizontally towards axis
			tipX = dotX - params.CrossAxisDir*halfDotSize*1.2
			tipY = dotY
		}
		points := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f", p1xArrow, p1yArrow, p2xArrow, p2yArrow, tipX, tipY)
		fmt.Fprintf(svg, `  <polygon points="%s" fill="%s"/>\n`, points, dotColor)
	}
	// Update bounds for the dot itself
	bounds.updateRect(dotX-halfDotSize, dotY-halfDotSize, dotSize, dotSize)
}

// Draw the year element with optional shape and link
func drawYearElement(svg *bytes.Buffer, bounds *bounds, entry TimelineEntry,
	yearStyle YearTextStyle, centerX, centerY float64) {
	yearStr := entry.Period
	yearWidth, yearHeight := estimateTextSVGWidth(yearStr, yearStyle.Font), getEstimatedHeight(yearStyle.Font)

	// --- Link Wrapper (around Year element) ---
	if entry.Link != "" {
		linkOpenTag := fmt.Sprintf(`<a xlink:href="%s" target="_blank">`, escapeXML(entry.Link))
		svg.WriteString("  " + linkOpenTag + "\n")
	}

	// Draw background shape
	shapeType, shapeParams, err := parseShapeString(yearStyle.Shape)
	if err != nil {
		log.Printf("Warning: Error parsing shape string \"%s\" for year \"%s\": %v. Skipping shape.",
			yearStyle.Shape, yearStr, err)
		shapeType = "none"
	}

	drawYearShape(svg, YearShapeParams{
		ShapeType:   shapeType,
		ShapeParams: shapeParams,
		CenterX:     centerX,
		CenterY:     centerY,
		TextWidth:   yearWidth,
		TextHeight:  yearHeight,
		YearStyle:   yearStyle,
	})

	// --- DEBUG LOGGING START ---
	// // log.Printf("DEBUG drawYearElement (%s): CenterX=%.2f, CenterY=%.2f, Color=%s, Size=%d, Family=%s",
	// // 	yearStr, centerX, centerY, yearStyle.TextColor, yearStyle.Font.FontSize, yearStyle.Font.FontFamily)
	// --- DEBUG LOGGING END ---

	// Draw the year text
	fmt.Fprintf(svg, `    <text x="%.2f" y="%.2f" font-family="%s" font-size="%d" font-weight="%s" font-style="%s" fill="%s" dominant-baseline="middle" text-anchor="middle">`,
		centerX, centerY, yearStyle.Font.FontFamily, yearStyle.Font.FontSize,
		yearStyle.Font.FontWeight, yearStyle.Font.FontStyle, yearStyle.TextColor)
	svg.WriteString(escapeXML(yearStr))
	svg.WriteString(`</text>`)
	svg.WriteString("\n")

	// Update bounds for text
	estWidth := math.Min(float64(len(yearStr))*float64(yearStyle.Font.FontSize)*0.7, 200)
	estHeight := float64(yearStyle.Font.FontSize)
	boundsX := centerX - estWidth/2.0
	boundsY := centerY - estHeight/2.0
	bounds.updateRect(boundsX, boundsY, estWidth, estHeight)

	// Close link wrapper
	if entry.Link != "" {
		svg.WriteString("  </a>\n")
	}
}

// Update the drawYearShape function to use the parameter struct
func drawYearShape(svg *bytes.Buffer, params YearShapeParams) {
	switch params.ShapeType {
	case "circle":
		radius := params.ShapeParams["r"]
		if radius < 0 { // Handle 'auto' radius
			// Calculate radius based on text dimensions + default internal padding
			const defaultAutoPadding = 4.0
			textHalfWidth := params.TextWidth / 2.0
			textHalfHeight := params.TextHeight / 2.0
			radius = math.Max(textHalfWidth, textHalfHeight) + defaultAutoPadding
			// Ensure minimum reasonable radius if text is tiny
			if radius < defaultAutoPadding*1.5 {
				radius = defaultAutoPadding * 1.5
			}
		} else if radius == 0 {
			// If radius is explicitly 0, draw nothing
			return
		}
		// Draw the circle
		fmt.Fprintf(svg, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" stroke="%s" stroke-width="%.2f"/>`,
			params.CenterX, params.CenterY, radius,
			params.YearStyle.FillColor, params.YearStyle.BorderColor, params.YearStyle.BorderWidth)
		svg.WriteString("\n")

	case "rectangle":
		rectW := params.ShapeParams["w"]
		rectH := params.ShapeParams["h"]
		if rectW > 0 && rectH > 0 {
			rectX := params.CenterX - rectW/2.0
			rectY := params.CenterY - rectH/2.0
			fmt.Fprintf(svg, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.2f"/>`,
				rectX, rectY, rectW, rectH,
				params.YearStyle.FillColor, params.YearStyle.BorderColor, params.YearStyle.BorderWidth)
			svg.WriteString("\n")
		}
	}
}

// Calculate the layout for a comment block
func calculateCommentBlockLayout(params CommentParams) CommentBlockLayout {
	layout := CommentBlockLayout{}

	// --- Parse Padding ---
	padTop, padRight, padBottom, padLeft := parsePadding(params.Style.Padding)
	layout.padTop, layout.padRight, layout.padBottom, layout.padLeft = padTop, padRight, padBottom, padLeft

	// --- Calculate Vertical Positions & Estimated Heights (Relative to Block Top - Inside Padding) ---
	currentRelY := padTop // Start drawing below top padding
	var titleLineRelY, titleTextRelY, bodyRelY float64
	var estTitleHeight, estLineHeight float64
	titleFont := params.Style.TitleFont
	titleLine := params.Style.TitleLine

	// Title Text Position & Height
	estTitleWidth := 0.0
	if params.TitleText != "" {
		titleTextRelY = currentRelY
		estTitleHeight = getEstimatedHeight(titleFont)
		estTitleWidth = estimateTextSVGWidth(params.TitleText, titleFont)
		currentRelY += estTitleHeight
	} else {
		estTitleHeight = 0
	}

	// Title Line Position & Height
	if titleLine.Visible && titleLine.Width > 0 && titleLine.Length > 0 {
		currentRelY += titleLine.Margin
		titleLineRelY = currentRelY
		estLineHeight = titleLine.Width
		currentRelY += estLineHeight
		currentRelY += titleLine.Margin
		// Consider title line length for width calculation
		estTitleWidth = math.Max(estTitleWidth, titleLine.Length)
	} else {
		estLineHeight = 0
	}

	// Body Position (foreignObject Y relative to Block Top)
	bodyRelY = currentRelY

	// Estimate foreignObject height (content only, no padding)
	layout.foHeight = calculateForeignObjectHeight(params.BodyText, params.ImageURL)

	// --- Calculate Visual Block Dimensions ---
	requiredContentWidth := estTitleWidth // Base width on title/line

	// Check for fixed block width from style
	if params.Style.BlockWidth != nil && *params.Style.BlockWidth > 0 {
		layout.contentWidth = *params.Style.BlockWidth // Use specified width for content
	} else {
		// Fallback: Use title/line width as content width (current behavior)
		layout.contentWidth = requiredContentWidth
	}

	// Ensure content width is not negative
	if layout.contentWidth < 0 {
		layout.contentWidth = 0
	}

	// Calculate visual block width including padding
	layout.visualBlockWidth = layout.contentWidth + padLeft + padRight

	// Calculate visual block height (unchanged)
	layout.visualBlockHeight = currentRelY + layout.foHeight + padBottom // Includes top padding, content, bottom padding

	// --- Calculate Block Position (Top-Left Corner of Visual Block) ---
	layout.blockX, layout.blockY = calculateBlockPosition(params.AnchorX, params.AnchorY,
		layout.visualBlockWidth, layout.visualBlockHeight,
		params.CrossAxisDir, params.IsHorizontal)

	// --- Calculate Absolute Content Positions (relative to SVG origin) ---
	layout.contentCenterX = layout.blockX + padLeft + layout.contentWidth/2.0
	layout.titleTextAbsY = layout.blockY + titleTextRelY
	layout.titleLineAbsY = layout.blockY + titleLineRelY
	layout.bodyAbsY = layout.blockY + bodyRelY
	layout.bodyAbsX = layout.blockX + padLeft // Body/FO starts after left padding

	return layout
}

// Calculate height needed for foreignObject content
func calculateForeignObjectHeight(bodyText, imageURL string) float64 {
	foHeight := foreignObjectHeightEstimate
	if bodyText == "" && imageURL == "" {
		foHeight = 0
	} else if bodyText == "" && imageURL != "" {
		foHeight = imagePlaceholderHeight + 10 // Rough estimate for image only
	}
	return foHeight
}

// Calculate the position of a comment block based on anchor and direction
func calculateBlockPosition(anchorX, anchorY, blockWidth, totalHeight, crossAxisDir float64, isHorizontal bool) (float64, float64) {
	var blockX, blockY float64

	if isHorizontal {
		blockX = anchorX - blockWidth/2.0 // Horizontal centering relative to anchorX
		if crossAxisDir < 0 {             // Block is ABOVE the anchor point (e.g., horizontal top)
			// Position block so its BOTTOM edge is at anchorY
			blockY = anchorY - totalHeight // Correct: Top edge = AnchorY - Full Height
		} else { // Block is BELOW the anchor point (e.g., horizontal bottom)
			// Position block so its TOP edge is at anchorY
			blockY = anchorY
		}
	} else {
		blockY = anchorY - totalHeight/2.0 // Vertical centering relative to anchorY
		if crossAxisDir < 0 {              // Block is LEFT of the anchor point (e.g., vertical left)
			// Position block so its RIGHT edge is at anchorX
			blockX = anchorX - blockWidth // Adjust based on total height
		} else { // Block is RIGHT of the anchor point (e.g., vertical right)
			// Position block so its LEFT edge is at anchorX
			blockX = anchorX
		}
	}

	return blockX, blockY
}

// Draw the background rectangle for a comment
func drawCommentBackground(svg *bytes.Buffer, bounds *bounds, style CommentTextStyle, layout CommentBlockLayout) {
	if style.Shape == "rectangle" {
		// Use the calculated visual block dimensions and position
		rectX := layout.blockX
		rectY := layout.blockY
		rectW := layout.visualBlockWidth
		rectH := layout.visualBlockHeight
		rectFill := style.FillColor
		if rectFill == "" {
			rectFill = "none"
		}
		rectBorderColor := style.BorderColor
		if rectBorderColor == "" {
			rectBorderColor = "none"
		}
		rectBorderWidth := float64(style.BorderWidth)
		if rectBorderWidth < 0 {
			rectBorderWidth = 0
		}
		rectBorderStyle := style.BorderStyle
		rectBorderDashArray := getStrokeDashArray(rectBorderStyle, int(rectBorderWidth))
		fmt.Fprintf(svg, `    <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.2f"%s rx="3" ry="3"/>`,
			rectX, rectY, rectW, rectH, rectFill, rectBorderColor, rectBorderWidth, rectBorderDashArray)
		svg.WriteString("\n")
		bounds.updateRect(rectX, rectY, rectW, rectH)
	}
}

// Add parameter structs for remaining functions with too many parameters
type CommentTitleParams struct {
	TitleText  string
	TitleFont  FontStyle
	TitleColor string
	Layout     CommentBlockLayout
}

type CommentBodyParams struct {
	Params    CommentParams
	BodyFont  FontStyle
	TextColor string
	Layout    CommentBlockLayout
}

// Update drawCommentTitle to use the parameter struct
func drawCommentTitle(svg *bytes.Buffer, bounds *bounds, params CommentTitleParams) {
	fmt.Fprintf(svg, `    <text x="%.2f" y="%.2f" font-family="%s" font-size="%d" font-weight="%s" font-style="%s" fill="%s" text-anchor="middle" dominant-baseline="hanging">`,
		params.Layout.contentCenterX, params.Layout.titleTextAbsY, params.TitleFont.FontFamily, params.TitleFont.FontSize,
		params.TitleFont.FontWeight, params.TitleFont.FontStyle, params.TitleColor)
	svg.WriteString(escapeXML(params.TitleText))
	svg.WriteString(`</text>`)
	svg.WriteString("\n")
	bounds.updatePoint(params.Layout.contentCenterX, params.Layout.titleTextAbsY) // Approximate bounds update
}

// Helper function to get MIME type from file extension
func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" { // Fallback if mime package doesn't know
		switch ext {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case ".gif":
			return "image/gif"
		case ".svg":
			return "image/svg+xml"
		// Add more common types if needed
		default:
			return "application/octet-stream" // Generic fallback
		}
	}
	return mimeType
}

// Update drawCommentBody to use the parameter struct and embed local images
func drawCommentBody(svg *bytes.Buffer, bounds *bounds, params CommentBodyParams) {
	// Use the calculated content width for the foreignObject
	contentWidth := params.Layout.contentWidth
	bounds.updateRect(params.Layout.bodyAbsX, params.Layout.bodyAbsY, contentWidth, params.Layout.foHeight)

	fmt.Fprintf(svg, `    <foreignObject x="%.2f" y="%.2f" width="%.2f" height="%.2f">`,
		params.Layout.bodyAbsX, params.Layout.bodyAbsY, contentWidth, params.Layout.foHeight)
	svg.WriteString("\n")
	fmt.Fprintf(svg, `        <div xmlns="http://www.w3.org/1999/xhtml">`)

	// Use text-align from style, default to center
	textAlign := params.Params.Style.TextAlign
	if textAlign == "" {
		textAlign = "center"
	}

	// Prepare style string outside Fprintf for clarity
	bodyStyle := fmt.Sprintf("color:%s; font-family:%s; font-size:%dpx; font-weight:%s; font-style:%s; text-align:%s;",
		params.TextColor, escapeXML(params.BodyFont.FontFamily), params.BodyFont.FontSize,
		escapeXML(params.BodyFont.FontWeight), escapeXML(params.BodyFont.FontStyle), textAlign)

	fmt.Fprintf(svg, `<div class="comment-html-content" style="%s">`, bodyStyle)

	if params.Params.ImageURL != "" {
		imgSrc := params.Params.ImageURL
		// Check if it's a likely file path (not URL or data URI)
		if !strings.HasPrefix(imgSrc, "http://") && !strings.HasPrefix(imgSrc, "https://") && !strings.HasPrefix(imgSrc, "data:") {
			log.Printf("Attempting to read and embed local image: %s", imgSrc)
			imgData, err := os.ReadFile(imgSrc)
			if err != nil {
				log.Printf("Warning: Could not read image file '%s': %v. Skipping image.", imgSrc, err)
				imgSrc = "" // Clear src if file read failed
			} else {
				mimeType := getMimeType(imgSrc)
				imgSrc = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData))
				log.Printf("Successfully embedded image '%s' as data URI.", params.Params.ImageURL)
			}
		}

		// Only output image tag if imgSrc is still valid
		if imgSrc != "" {
			fmt.Fprintf(svg, `<img src="%s" style="max-width: 100%%; height: auto; display: block; margin-bottom: 5px;" alt="Timeline image"/>`,
				escapeXML(imgSrc)) // Escape the potentially long data URI? Probably not needed for src attribute.
			svg.WriteString("\n")
		}
	}

	if params.Params.BodyText != "" {
		// Basic markdown link support: [text](url)
		re := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`) // Escaped brackets
		formattedText := re.ReplaceAllString(params.Params.BodyText, `<a href="$2" target="_blank">$1</a>`)
		formattedText = strings.ReplaceAll(formattedText, "\n", "<br />") // Handle newlines
		svg.WriteString(formattedText)
		svg.WriteString("\n")
	}

	svg.WriteString(`</div></div>`)
	svg.WriteString("\n")
	svg.WriteString(`    </foreignObject>`)
	svg.WriteString("\n")
}

// Assemble the final SVG document
func assembleFinalSVG(svgBody bytes.Buffer, timelineBounds bounds, layoutPadding float64, globalFont *FontStyle) string {

	// --- DEBUG LOGGING START ---
	// log.Printf("--- Debug assembleFinalSVG ---")
	// if timelineBounds.isSet {
	// 	log.Printf("  Bounds: minX=%.2f, maxX=%.2f, minY=%.2f, maxY=%.2f",
	// 		timelineBounds.minX, timelineBounds.maxX, timelineBounds.minY, timelineBounds.maxY)
	// } else {
	// 	log.Printf("  Bounds: Not set")
	// }
	// --- DEBUG LOGGING END ---

	finalWidth := layoutPadding * 2
	finalHeight := layoutPadding * 2
	offsetX := layoutPadding - timelineBounds.minX
	offsetY := layoutPadding - timelineBounds.minY

	if timelineBounds.isSet {
		finalWidth += timelineBounds.maxX - timelineBounds.minX
		finalHeight += timelineBounds.maxY - timelineBounds.minY
	} else {
		finalWidth += 600 // Default size if bounds not set
		finalHeight += 100
	}

	finalWidth = math.Max(finalWidth, 10)
	finalHeight = math.Max(finalHeight, 10)

	// --- DEBUG LOGGING START ---
	// log.Printf("  Calculated: finalWidth=%.0f, finalHeight=%.0f, offsetX=%.2f, offsetY=%.2f",
	// 	finalWidth, finalHeight, offsetX, offsetY)
	// --- DEBUG LOGGING END ---

	var finalSVG bytes.Buffer
	fmt.Fprintf(&finalSVG, `<svg width="%.0f" height="%.0f" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`,
		finalWidth, finalHeight)
	finalSVG.WriteString("\n")

	// Add a white background rectangle
	fmt.Fprintf(&finalSVG, `  <rect width="%.0f" height="%.0f" fill="#FFFFFF" />\n`, finalWidth, finalHeight)

	// Styles - Keep the tags but remove the placeholder comment
	finalSVG.WriteString("  <style>\n")
	if globalFont != nil { /* Placeholder for potential future global font CSS */
	}
	finalSVG.WriteString("  </style>\n")

	// Transform Group...
	fmt.Fprintf(&finalSVG, `<g transform="translate(%.2f, %.2f)">`, offsetX, offsetY)
	finalSVG.WriteString("\n")
	finalSVG.Write(svgBody.Bytes())
	finalSVG.WriteString("</g>\n")
	finalSVG.WriteString("</svg>")

	return finalSVG.String()
}

// Helper: Draw Junction Marker
func drawJunctionMarker(svg *bytes.Buffer, bounds *bounds, params JunctionMarkerParams) {
	if params.Style.Shape == "none" || params.Style.Size <= 0 {
		return
	}
	size := params.Style.Size
	halfSize := size / 2.0
	fillColor := params.MarkerColor
	switch params.Style.Shape {
	case "arrow", "diamond": /* ... draw polygons ... */
		var points1, points2 string
		var p1x, p1y, p2x, p2y, p3x, p3y, p4x, p4y float64
		if params.IsHorizontal {
			p2x, p2y = params.CenterX-halfSize, params.CenterY
			p3x, p3y = params.CenterX+halfSize, params.CenterY
			p1x, p1y = params.CenterX, params.CenterY+halfSize
			p4x, p4y = params.CenterX, params.CenterY-halfSize
		} else {
			p2x, p2y = params.CenterX, params.CenterY-halfSize
			p3x, p3y = params.CenterX, params.CenterY+halfSize
			p1x, p1y = params.CenterX+halfSize, params.CenterY
			p4x, p4y = params.CenterX-halfSize, params.CenterY
		}
		points1 = fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f", p1x, p1y, p2x, p2y, p3x, p3y)
		points2 = fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f", p4x, p4y, p2x, p2y, p3x, p3y)
		fmt.Fprintf(svg, `  <polygon points="%s" fill="%s" />`, points1, fillColor)
		fmt.Fprintf(svg, `  <polygon points="%s" fill="%s" />`, points2, fillColor)
		svg.WriteString("\n")
		bounds.updatePoint(params.CenterX-halfSize, params.CenterY-halfSize)
		bounds.updatePoint(params.CenterX+halfSize, params.CenterY+halfSize)
	case "circle": /* ... draw circle ... */
		fmt.Fprintf(svg, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" />`,
			params.CenterX, params.CenterY, halfSize, fillColor)
		svg.WriteString("\n")
		bounds.updateRect(params.CenterX-halfSize, params.CenterY-halfSize, size, size)
	}
}

// Helper: Draw Comment
func drawComment(svg *bytes.Buffer, bounds *bounds, params CommentParams) {
	// --- Font and Color Setup ---
	bodyFont := params.Style.Font
	titleFont := params.Style.TitleFont

	titleLine := params.Style.TitleLine
	textColor := params.Style.TextColor
	if textColor == "" {
		textColor = params.DefaultColor
	}
	titleColor := params.Style.TitleColor
	if titleColor == "" {
		titleColor = textColor // Title defaults to body text color
	}

	// --- Block Layout Calculation ---
	blockLayout := calculateCommentBlockLayout(params)

	// --- Draw Background/Border ---
	drawCommentBackground(svg, bounds, params.Style, blockLayout)

	// --- Draw Title Text ---
	if params.TitleText != "" {
		drawCommentTitle(svg, bounds, CommentTitleParams{
			TitleText:  params.TitleText,
			TitleFont:  titleFont,
			TitleColor: titleColor,
			Layout:     blockLayout,
		})
	}

	// --- Draw Title Line (Decorative) ---
	if params.TitleText != "" && titleLine.Visible && titleLine.Length > 0 && titleLine.Width > 0 {
		drawCommentTitleLine(svg, bounds, CommentTitleLineParams{
			TitleLine: titleLine,
			Layout:    blockLayout,
		})
	}

	// --- Draw Body Content (foreignObject) ---
	if blockLayout.foHeight > 0 {
		drawCommentBody(svg, bounds, CommentBodyParams{
			Params:    params,
			BodyFont:  bodyFont,
			TextColor: textColor,
			Layout:    blockLayout,
		})
	}
}

// Helper: Calculate Element Center
func calculateElementCenter(params ElementCenterParams) (float64, float64) {
	centerX, centerY := params.AxisX, params.AxisY // Start at the entry point on axis
	if params.IsHorizontal {                       // Base orientation is horizontal
		// MainOffset shifts along the intended horizontal axis (X) - usually 0 for these elements
		centerX += params.MainOffset
		// CrossOffset shifts vertically (Y) based on CrossDir (+1 down, -1 up)
		// ConnectorLen provides the base distance from the axis.
		centerY += params.CrossDir * (params.ConnectorLen + params.CrossOffset)
	} else { // Base orientation is vertical
		// MainOffset shifts along the intended vertical axis (Y) - usually 0
		centerY += params.MainOffset
		// CrossOffset shifts horizontally (X) based on CrossDir (+1 right, -1 left)
		centerX += params.CrossDir * (params.ConnectorLen + params.CrossOffset)
	}
	return centerX, centerY
}

// Add a parameter struct for drawCommentTitleLine
type CommentTitleLineParams struct {
	TitleLine TitleLineStyle
	Layout    CommentBlockLayout
}

// Draw the decorative line below a comment title
func drawCommentTitleLine(svg *bytes.Buffer, bounds *bounds, params CommentTitleLineParams) {
	lineX1 := params.Layout.contentCenterX - params.TitleLine.Length/2.0
	lineX2 := params.Layout.contentCenterX + params.TitleLine.Length/2.0
	fmt.Fprintf(svg, ` <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f" />`,
		lineX1, params.Layout.titleLineAbsY, lineX2, params.Layout.titleLineAbsY, params.TitleLine.Color, params.TitleLine.Width)
	svg.WriteString("\n")
	bounds.updatePoint(lineX1, params.Layout.titleLineAbsY)
	bounds.updatePoint(lineX2, params.Layout.titleLineAbsY)
}

// Helper function to draw a single segment of the center line
func drawCenterLineSegment(params DrawCenterLineSegmentParams) {
	strokeDash := getStrokeDashArray(params.LineType, int(params.Width))
	strokeLineCap := ""
	if params.RoundedCaps {
		strokeLineCap = ` stroke-linecap="round"`
	}

	fmt.Fprintf(params.SVG, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s%s />`+"\n",
		params.X1, params.Y1, params.X2, params.Y2, params.Color, params.Width, strokeDash, strokeLineCap)
	params.Bounds.updatePoint(params.X1, params.Y1)
	params.Bounds.updatePoint(params.X2, params.Y2)
}

// Helper function to draw a single axis segment and update current coordinates
// Returns the end coordinates (new currentX, new currentY) of the drawn segment.
func drawAndAdvanceAxisSegment(params DrawAndAdvanceAxisSegmentParams) (float64, float64) {
	var segmentLength float64
	var angleOverride *float64
	var segmentColorIndex int

	if params.StartIndex == -1 { // Initial segment (from 0 to first junction)
		segmentLength = params.Data.junctionPoints[0]
		if len(params.Entries) > 0 {
			angleOverride = params.Entries[0].AngleOverride
		}
		segmentColorIndex = 0
	} else { // Subsequent segment (from junction i to junction i+1)
		segmentLength = params.Data.junctionPoints[params.StartIndex+1] - params.Data.junctionPoints[params.StartIndex]
		if params.StartIndex+1 < len(params.Entries) {
			angleOverride = params.Entries[params.StartIndex+1].AngleOverride
		}
		segmentColorIndex = params.StartIndex + 1
	}

	// Calculate geometry
	segStartX, segStartY, segEndX, segEndY, _ := calculateAxisGeometry(
		params.CurrentX, params.CurrentY, segmentLength, params.BaseOrientation,
		params.GlobalAxisAngle, angleOverride,
	)

	// Determine color
	drawColor := params.Data.segmentColors[segmentColorIndex]
	if drawColor == "" {
		drawColor = params.LayoutConfig.centerLineBaseColor
	}

	// Draw the segment
	drawCenterLineSegment(DrawCenterLineSegmentParams{
		SVG:         params.SVG,
		Bounds:      params.Bounds,
		X1:          segStartX,
		Y1:          segStartY,
		X2:          segEndX,
		Y2:          segEndY,
		Color:       drawColor,
		Width:       params.LayoutConfig.centerLineWidth,
		LineType:    params.CenterLineType,
		RoundedCaps: params.LayoutConfig.centerLineIsRounded,
	})

	// Return the end coordinates for the next iteration
	return segEndX, segEndY
}

// GenerateSVG generates an SVG timeline from a template and entries
func GenerateSVG(template Template, entries []TimelineEntry) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("no timeline entries to generate")
	}

	var svgBody bytes.Buffer
	timelineBounds := bounds{}
	isHorizontal := template.CenterLine.Orientation == "horizontal"

	layoutConfig := initializeLayoutConfig(template)
	timelineData := calculateTimelinePositionsAndStyles(entries, template, layoutConfig)

	startX, startY := 0.0, 0.0
	timelineBounds.updatePoint(startX, startY)

	// --- Phase 1: Pre-calculate all axis geometry ---
	type AxisPoint struct {
		X, Y float64
	}
	entryAxisPoints := make([]AxisPoint, len(entries))
	segmentStartPoints := make([]AxisPoint, len(entries)) // Start point of segment LEADING to entry i
	segmentEndPoints := make([]AxisPoint, len(entries))   // End point of segment LEADING to entry i ( = start of next)

	currentX, currentY := startX, startY
	globalAxisAngle := template.CenterLine.Angle
	baseOrientation := template.CenterLine.Orientation

	// Calculate geometry for the initial segment (before first entry)
	initialSegStartX, initialSegStartY, initialSegEndX, initialSegEndY, _ := calculateAxisGeometry(
		currentX, currentY, timelineData.junctionPoints[0], // Length is from 0 to first junction
		baseOrientation, globalAxisAngle,
		entries[0].AngleOverride, // Use first entry's override for the first segment
	)
	currentX, currentY = initialSegEndX, initialSegEndY // Update position to end of first segment (start of first entry)

	// Calculate geometry for segments between entries and entry points
	for i := range entries {
		// Store the axis point for this entry (which is the end of the previous segment)
		entryAxisPoints[i] = AxisPoint{X: currentX, Y: currentY}

		// Calculate the segment that *follows* this entry (if not the last)
		if i < len(entries)-1 {
			segmentLength := timelineData.junctionPoints[i+1] - timelineData.junctionPoints[i]
			var nextAngleOverride *float64
			if i+1 < len(entries) {
				nextAngleOverride = entries[i+1].AngleOverride
			}

			segStartX, segStartY, segEndX, segEndY, _ := calculateAxisGeometry(
				currentX, currentY, segmentLength,
				baseOrientation, globalAxisAngle, nextAngleOverride,
			)
			// Store segment start/end points (relative to the *following* entry)
			segmentStartPoints[i+1] = AxisPoint{X: segStartX, Y: segStartY}
			segmentEndPoints[i+1] = AxisPoint{X: segEndX, Y: segEndY}

			currentX, currentY = segEndX, segEndY // Advance position
		}
	}
	// Need start/end for the very first segment separately
	segmentStartPoints[0] = AxisPoint{X: initialSegStartX, Y: initialSegStartY}
	segmentEndPoints[0] = AxisPoint{X: initialSegEndX, Y: initialSegEndY}

	// --- Phase 2: Draw all Center Line Segments FIRST ---
	centerLineType := template.CenterLine.Type
	for i := range entries {
		drawColor := timelineData.segmentColors[i]
		if drawColor == "" {
			drawColor = layoutConfig.centerLineBaseColor
		}
		drawCenterLineSegment(DrawCenterLineSegmentParams{
			SVG:         &svgBody,
			Bounds:      &timelineBounds,
			X1:          segmentStartPoints[i].X,
			Y1:          segmentStartPoints[i].Y,
			X2:          segmentEndPoints[i].X,
			Y2:          segmentEndPoints[i].Y,
			Color:       drawColor,
			Width:       layoutConfig.centerLineWidth,
			LineType:    centerLineType,
			RoundedCaps: layoutConfig.centerLineIsRounded,
		})
	}

	// --- Phase 3: Draw all Entries ON TOP ---
	for i, entry := range entries {
		// Use the pre-calculated axis point for this entry
		drawTimelineEntry(&svgBody, &timelineBounds, TimelineEntryParams{
			Index:        i,
			Entry:        entry,
			Data:         timelineData,
			EntryAxisX:   entryAxisPoints[i].X,
			EntryAxisY:   entryAxisPoints[i].Y,
			IsHorizontal: isHorizontal,
			Config:       layoutConfig,
		})
	}

	return assembleFinalSVG(svgBody, timelineBounds, layoutConfig.layoutPadding, template.GlobalFont), nil
}
