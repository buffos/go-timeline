# Timeline Generator JSON Schema Definition

This document outlines the structure for the `template.json` and `data.json` files used by the timeline generator.

## Template (`template.json`)

Defines the overall appearance and default styles.

```json
{
  "center_line": {
    // Defines the main axis of the timeline
    "width": "number (pixels, default: 2)",
    "type": "string ('solid'|'dotted'|'dashed', default: 'solid')",
    "orientation": "string ('horizontal'|'vertical', required)",
    "angle": "number (Optional, degrees, overrides orientation for axis angle, 0=right, 90=up)",
    "color": "string (CSS color, default: '#000000')",
    "rounded_caps": "boolean (default: false, use rounded line endings)"
  },
  "layout": {
    // Global layout settings
    "padding": "number (pixels, default: 50, overall padding around SVG content)",
    "entry_spacing": "number (pixels, default: 150, spacing between entry centers)",
    "connector_length": "number (pixels, default: 50, default distance from center line)"
  },
  "global_font": {
    // Optional: Default font settings for all text elements unless overridden
    "font_family": "string (CSS font-family, default: 'Arial, sans-serif')",
    "font_size": "number (pixels, default: 12)",
    "font_weight": "string (CSS font-weight, default: 'normal')",
    "font_style": "string ('normal'|'italic', default: 'normal')"
  },
  "period_defaults": {
    // Default styles for each timeline entry's components
    "year_text": {
      "position": "string ('start'|'end'|'alternate-start-end'|'alternate-end-start', default: 'start')",
      "main_axis_offset": "number (pixels, default: 0, offset along the main timeline axis)",
      "cross_axis_offset": "number (pixels, default: 0, offset perpendicular to the timeline axis, added to connector_length)",
      "font": {
        "font_family": "string (inherits global_font or default)",
        "font_size": "number (pixels, inherits global_font or default: 14)",
        "font_weight": "string (inherits global_font or default: 'bold')",
        "font_style": "string (inherits global_font or default: 'normal')"
      },
      "text_color": "string (CSS color, default: '#000000')",
      "shape": "string (e.g., 'none', 'circle;r=10', 'rectangle;w=40;h=20', default: 'circle;r=auto'). If 'auto', radius is based on text size.",
      "fill_color": "string (CSS color, default: '#FFFFFF')",
      "border_color": "string (CSS color, default: connector color)",
      "border_width": "number (pixels, default: 1.5)"
    },
    "connector": {
      "color": "string (CSS color, default: '#888888')",
      "line_type": "string ('solid'|'dotted'|'dashed', default: 'solid')",
      "width": "number (pixels, default: 1)",
      "side": "string (Optional, 'top'/'bottom' for horizontal, 'left'/'right' for vertical, overrides default alternating behavior)",
      "draw_to_period": "boolean (default: true, draw line from axis to period element)",
      "draw_to_comment": "boolean (default: true, draw line from axis to comment element)",
      "dot": { // Configuration for the dot drawn on the connector
        "size": "number (pixels, diameter, default: 8)",
        "color": "string (CSS color, default: connector color)",
        "shape": "string ('circle'|'square'|'arrow'|'none', default: 'circle')",
        "visible": "boolean (default: true)",
        "offset_main": "number (pixels, offset along the connector line from the endpoint, default: 0)",
        "offset_cross": "number (pixels, offset perpendicular to the connector line, default: 0)",
        "stop_at_dot": "boolean (default: true, connector line stops at dot position)"
      }
    },
    "comment_text": {
      "position": "string ('start'|'end'|'alternate-start-end'|'alternate-end-start', default: 'alternate-start-end')",
      "main_axis_offset": "number (Optional, pixels, default: 0, adjusts position parallel to the timeline axis)",
      "cross_axis_offset": "number (Optional, pixels, default: 0, adjusts distance from axis perpendicular to orientation)",
      "font": {
        // Font for the body text
        "font_family": "string (inherits global_font or default)",
        "font_size": "number (pixels, inherits global_font or default: 12)",
        "font_weight": "string (inherits global_font or default: 'normal')",
        "font_style": "string (inherits global_font or default: 'normal')"
      },
      "title_font": {
        // Specific font style for the title (defaults to body font)
        "font_family": "string (inherits body font or default)",
        "font_size": "number (pixels, inherits body font or default)",
        "font_weight": "string (inherits body font or default: 'bold')",
        "font_style": "string (inherits body font or default: 'normal')"
      },
      "title_line": {
        // Decorative line above title
        "visible": "boolean (default: true if width/length > 0)",
        "color": "string (CSS color, defaults to connector color)",
        "width": "number (pixels, thickness, default: 1)",
        "length": "number (pixels, length, default: 30)",
        "margin": "number (pixels, space applied both above and below the title line, default: 4)"
      },
      "title_color": "string (CSS color, defaults to body text_color)",
      "shape": "string ('rectangle'|'none', default: 'rectangle')",
      "fill_color": "string (CSS color, default: '#f8f8f8')",
      "text_color": "string (CSS color, default: '#333333', for body)",
      "padding": "string (CSS-style: e.g., \"8\", \"10 20\", \"5 10 15 20\", default: \"8\")",
      "block_width": "number (Optional, pixels), specifies a fixed width for the content area (foreignObject). If omitted or <= 0, width is estimated based on title/line length.",
      "border_color": "string (CSS color, default: '#dddddd')",
      "border_width": "number (pixels, default: 1)",
      "border_style": "string ('solid'|'dotted'|'dashed', default: 'solid')",
      "text_align": "string ('left'|'center'|'right', default: 'center', applies within comment block)"
    },
    "centerline_projection": {
      // Style for the segment on the main center line for this entry
      "color": "string (CSS color, default: center_line.color)"
    },
    "junction_marker": {
      // Marker placed at the entry's center point on the main axis
      "shape": "string ('diamond'|'arrow'|'circle'|'none', default: 'circle')",
      "size": "number (pixels, default: 8)",
      "color": "string (CSS color, optional, defaults derived from segment/connector)"
    }
  }
}
```

## Data (`data.json`)

Contains the actual timeline events.

```json
{
  "entries": [
    {
      "period": "string (Required, label for the entry, e.g., year)",
      "title_text": "string (Optional, title for the comment block)",
      "comment_text": "string (Optional, body text/HTML for the comment block, use '\\n' for newlines)",
      "comment_image": "string (Optional, URL or local path for an image in the comment block)",
      "link": "string (Optional, URL to link the period element to)",
      "entry_spacing_override": "number (Optional, pixels, overrides layout.entry_spacing *after* this entry)",
      "orientation_override": "string (Optional, 'horizontal' or 'vertical', overrides center_line.orientation for annotation placement for this entry)",
      "angle_override": "number (Optional, degrees, overrides center_line.angle for this entry's segment)",

      // --- Overrides (all optional, fields within are optional) ---
      "year_text_override": {
        "position": "string ('start'|'end'|'alternate-start-end'|'alternate-end-start')",
        "main_axis_offset": "number (Optional, pixels)",
        "cross_axis_offset": "number (Optional, pixels)",
        "font": {
          "font_family": "string",
          "font_size": "number",
          "font_weight": "string",
          "font_style": "string ('normal'|'italic')"
        },
        "text_color": "string",
        "shape": "string (e.g., 'none', 'circle;r=10', 'rectangle;w=40;h=20', default: 'circle;r=auto'). If 'auto', radius is based on text size.",
        "fill_color": "string",
        "border_color": "string",
        "border_width": "number"
      },
      "connector_override": {
        "color": "string",
        "line_type": "string ('solid'|'dotted'|'dashed')",
        "width": "number",
        "side": "string (Optional, 'top'/'bottom'/'left'/'right')",
        "draw_to_period": "boolean",
        "draw_to_comment": "boolean",
        "dot": { // Override for the dot drawn on the connector
          "size": "number",
          "color": "string",
          "shape": "string ('circle'|'square'|'arrow'|'none')",
          "visible": "boolean",
          "offset_main": "number",
          "offset_cross": "number",
          "stop_at_dot": "boolean"
        }
      },
      "comment_text_override": {
        "position": "string ('start'|'end'|'alternate-start-end'|'alternate-end-start')",
        "main_axis_offset": "number (Optional, pixels)",
        "cross_axis_offset": "number (Optional, pixels)",
        "font": { // Body font overrides
          "font_family": "string",
          "font_size": "number",
          "font_weight": "string",
          "font_style": "string ('normal'|'italic')"
        },
        "title_font": { // Title font overrides
          "font_family": "string",
          "font_size": "number",
          "font_weight": "string",
          "font_style": "string ('normal'|'italic')"
        },
        "title_line": {
          "visible": "boolean",
          "color": "string",
          "width": "number",
          "length": "number",
          "margin": "number"
        },
        "title_color": "string",
        "shape": "string ('rectangle'|'none')",
        "fill_color": "string",
        "text_color": "string", // Body text color
        "padding": "string",
        "block_width": "number (Optional, pixels)",
        "border_color": "string",
        "border_width": "number",
        "border_style": "string ('solid'|'dotted'|'dashed')",
        "text_align": "string ('left'|'center'|'right')"
      },
      "centerline_projection_override": {
        "color": "string"
      },
      "junction_marker_override": {
        "shape": "string ('diamond'|'arrow'|'circle'|'none')",
        "size": "number",
        "color": "string"
      }
    }
  ]
}
