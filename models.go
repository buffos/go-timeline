package main

// --- Template Structs ---

// Added: Global layout configurations
type LayoutOptions struct {
	Padding         float64 `json:"padding"`          // Overall padding around the timeline content
	EntrySpacing    float64 `json:"entry_spacing"`    // Default spacing between entry centers
	ConnectorLength float64 `json:"connector_length"` // Default connector length
	// Add other global layout defaults here if needed
}

// JunctionMarkerStyle defines the marker between timeline segments
type JunctionMarkerStyle struct {
	Shape string  `json:"shape"` // "diamond", "arrow", "none"
	Size  float64 `json:"size"`  // Size of the marker (e.g., width/height)
	// Color is typically derived from the next segment, but can be overridden
	Color *string `json:"color,omitempty"`
}

// TitleLineStyle defines the decorative line above comment titles
type TitleLineStyle struct {
	Visible bool    `json:"visible"` // Default false? Or based on width/length? Let's default true if width/length > 0
	Color   string  `json:"color"`   // Defaults to segment/connector color
	Width   float64 `json:"width"`   // Thickness
	Length  float64 `json:"length"`  // Length
	Margin  float64 `json:"margin"`  // Space below the line, above the title
}

// FontStyle defines common font properties
type FontStyle struct {
	FontFamily string `json:"font_family,omitempty"`
	FontSize   int    `json:"font_size,omitempty"`   // Use int for pixels initially
	FontWeight string `json:"font_weight,omitempty"` // e.g., "normal", "bold", "400", "700"
	FontStyle  string `json:"font_style,omitempty"`  // "normal", "italic"
}

type Template struct {
	CenterLine     CenterLine    `json:"center_line"`
	Layout         LayoutOptions `json:"layout"`
	GlobalFont     *FontStyle    `json:"global_font,omitempty"` // Added Global Font Defaults (pointer)
	PeriodDefaults PeriodStyle   `json:"period_defaults"`
}

type CenterLine struct {
	Width       int      `json:"width"`
	Type        string   `json:"type"`
	Orientation string   `json:"orientation"`
	Angle       *float64 `json:"angle,omitempty"` // Added: Optional angle in degrees
	Color       string   `json:"color"`
	RoundedCaps bool     `json:"rounded_caps"` // Added for rounded ends
}

type PeriodStyle struct {
	YearText             YearTextStyle             `json:"year_text"`
	Connector            ConnectorStyle            `json:"connector"` // Still needed for line style/color
	CommentText          CommentTextStyle          `json:"comment_text"`
	CenterlineProjection CenterlineProjectionStyle `json:"centerline_projection"`
	JunctionMarker       JunctionMarkerStyle       `json:"junction_marker"` // Added Junction Marker default
}

type YearTextStyle struct {
	Position        string    `json:"position,omitempty"`          // Optional placement override
	MainAxisOffset  float64   `json:"main_axis_offset,omitempty"`  // Added back
	CrossAxisOffset float64   `json:"cross_axis_offset,omitempty"` // Added back
	TextColor       string    `json:"text_color,omitempty"`
	Font            FontStyle `json:"font,omitempty"`
	Shape           string    `json:"shape,omitempty"`
	FillColor       string    `json:"fill_color,omitempty"`
	BorderColor     string    `json:"border_color,omitempty"`
	BorderWidth     float64   `json:"border_width,omitempty"`
}

type ConnectorStyle struct {
	DrawToPeriod  *bool    `json:"draw_to_period,omitempty"`
	DrawToComment *bool    `json:"draw_to_comment,omitempty"`
	Width         int      `json:"width,omitempty"`
	Color         string   `json:"color,omitempty"`
	LineType      string   `json:"line_type,omitempty"`
	Side          string   `json:"side,omitempty"` // Added
	Dot           DotStyle `json:"dot,omitempty"`
}

type DotStyle struct {
	Size        int    `json:"size"` // diameter
	Color       string `json:"color"`
	Shape       string `json:"shape"` // "circle", "arrow", "square", "none"
	Visible     bool   `json:"visible"`
	OffsetMain  int    `json:"offset_main"`  // Offset along the connector line
	OffsetCross int    `json:"offset_cross"` // Offset perpendicular to the connector line
	StopAtDot   bool   `json:"stop_at_dot"`  // Added: Control if line stops at dot
}

type CommentTextStyle struct {
	Position        string         `json:"position"`
	MainAxisOffset  float64        `json:"main_axis_offset,omitempty"` // Added back
	CrossAxisOffset float64        `json:"cross_axis_offset,omitempty"`
	Font            FontStyle      `json:"font"`        // Font for the body text
	TitleFont       FontStyle      `json:"title_font"`  // Added: Specific font style for the title
	TitleLine       TitleLineStyle `json:"title_line"`  // Added: Decorative line above title
	TitleColor      string         `json:"title_color"` // Added: Specific color for the title text
	Shape           string         `json:"shape"`       // "rectangle", "none" - determines background/border for body
	FillColor       string         `json:"fill_color"`
	TextColor       string         `json:"text_color"`            // Color for the body text
	Padding         string         `json:"padding"`               // Changed: Padding string (e.g., "10", "10 20", "10 20 30 40")
	BlockWidth      *float64       `json:"block_width,omitempty"` // Added: Optional fixed width
	BorderColor     string         `json:"border_color"`
	BorderWidth     int            `json:"border_width"`
	BorderStyle     string         `json:"border_style"`
	TextAlign       string         `json:"text_align"` // Added: Alignment for text within comment block ('left', 'center', 'right')
}

// Added: Style for the segment on the main center line corresponding to a period
type CenterlineProjectionStyle struct {
	Color string `json:"color"`
	// Percentage float64 `json:"percentage"` // Deferring variable length percentage, assume equal spacing for now
}

// --- Data Structs ---

type TimelineData struct {
	Entries []TimelineEntry `json:"entries"`
}

type TimelineEntry struct {
	Period                       string                     `json:"period"`                 // Used as year text if no shape, or inside shape
	TitleText                    string                     `json:"title_text,omitempty"`   // Optional Title for comment section
	CommentText                  string                     `json:"comment_text,omitempty"` // Body text for comment section
	CommentImage                 string                     `json:"comment_image,omitempty"`
	Link                         string                     `json:"link,omitempty"` // Applied to Period/Year element
	EntrySpacingOverride         *float64                   `json:"entry_spacing_override,omitempty"`
	OrientationOverride          *string                    `json:"orientation_override,omitempty"` // Added
	AngleOverride                *float64                   `json:"angle_override,omitempty"`       // Added: Optional angle override in degrees
	ConnectorOverride            *ConnectorStyleOverride    `json:"connector_override,omitempty"`
	CommentTextOverride          *CommentTextStyleOverride  `json:"comment_text_override,omitempty"`
	YearTextOverride             *YearTextStyleOverride     `json:"year_text_override,omitempty"`
	CenterlineProjectionOverride *CenterlineProjectionStyle `json:"centerline_projection_override,omitempty"`
	JunctionMarkerOverride       *JunctionMarkerOverride    `json:"junction_marker_override,omitempty"`
}

// FontStyleOverride allows overriding individual font properties
type FontStyleOverride struct {
	FontFamily *string `json:"font_family,omitempty"`
	FontSize   *int    `json:"font_size,omitempty"`
	FontWeight *string `json:"font_weight,omitempty"`
	FontStyle  *string `json:"font_style,omitempty"`
}

type YearTextStyleOverride struct {
	Position        *string            `json:"position,omitempty"`
	MainAxisOffset  *float64           `json:"main_axis_offset,omitempty"`  // Added back
	CrossAxisOffset *float64           `json:"cross_axis_offset,omitempty"` // Added back
	Font            *FontStyleOverride `json:"font,omitempty"`
	TextColor       *string            `json:"text_color,omitempty"`
	Shape           *string            `json:"shape,omitempty"`        // Added
	FillColor       *string            `json:"fill_color,omitempty"`   // Added
	BorderColor     *string            `json:"border_color,omitempty"` // Added
	BorderWidth     *float64           `json:"border_width,omitempty"` // Added
}

type CommentTextStyleOverride struct {
	Position        *string                 `json:"position,omitempty"`
	MainAxisOffset  *float64                `json:"main_axis_offset,omitempty"` // Added back
	CrossAxisOffset *float64                `json:"cross_axis_offset,omitempty"`
	Font            *FontStyleOverride      `json:"font,omitempty"`        // Body font
	TitleFont       *FontStyleOverride      `json:"title_font,omitempty"`  // Title font override
	TitleLine       *TitleLineStyleOverride `json:"title_line,omitempty"`  // Title line override
	TitleColor      *string                 `json:"title_color,omitempty"` // Title text color override
	Shape           *string                 `json:"shape,omitempty"`
	FillColor       *string                 `json:"fill_color,omitempty"`
	TextColor       *string                 `json:"text_color,omitempty"`  // Body text color
	Padding         *string                 `json:"padding,omitempty"`     // Changed: Padding string override
	BlockWidth      *float64                `json:"block_width,omitempty"` // Added
	BorderColor     *string                 `json:"border_color,omitempty"`
	BorderWidth     *int                    `json:"border_width,omitempty"`
	BorderStyle     *string                 `json:"border_style,omitempty"`
	TextAlign       *string                 `json:"text_align,omitempty"` // Added
}

type JunctionMarkerOverride struct { // New Override Struct
	Shape *string  `json:"shape,omitempty"`
	Size  *float64 `json:"size,omitempty"`
	Color *string  `json:"color,omitempty"`
}

type TitleLineStyleOverride struct { // New Override Struct
	Visible *bool    `json:"visible,omitempty"`
	Color   *string  `json:"color,omitempty"`
	Width   *float64 `json:"width,omitempty"`
	Length  *float64 `json:"length,omitempty"`
	Margin  *float64 `json:"margin,omitempty"`
}

// Added: Override struct for ConnectorStyle to handle pointers
type ConnectorStyleOverride struct {
	Color         *string           `json:"color,omitempty"`
	LineType      *string           `json:"line_type,omitempty"`
	Width         *int              `json:"width,omitempty"`
	DrawToPeriod  *bool             `json:"draw_to_period,omitempty"`
	DrawToComment *bool             `json:"draw_to_comment,omitempty"`
	Dot           *DotStyleOverride `json:"dot,omitempty"` // Added missing Dot field
}

// Added: Override struct for DotStyle
type DotStyleOverride struct {
	Size        *int    `json:"size,omitempty"`
	Color       *string `json:"color,omitempty"`
	Shape       *string `json:"shape,omitempty"`
	Visible     *bool   `json:"visible,omitempty"`
	OffsetMain  *int    `json:"offset_main,omitempty"`
	OffsetCross *int    `json:"offset_cross,omitempty"`
	StopAtDot   *bool   `json:"stop_at_dot,omitempty"` // Added override
}
