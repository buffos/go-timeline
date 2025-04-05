# Go Timeline Generator

A command-line tool written in Go to generate customizable SVG timelines from JSON configuration files. It can also render these SVGs into PNG or JPG images using a headless browser.

## Features

*   **SVG Generation:** Creates clean, scalable SVG timelines.
*   **PNG/JPG Output:** Renders SVGs to raster image formats (PNG, JPG) via `chromedp`.
*   **Highly Customizable:** Control layout, colors, fonts, shapes, connector styles, offsets, and more through a JSON template.
*   **Data-Driven:** Define timeline entries (periods, titles, comments, images) in a separate JSON data file.
*   **Style Overrides:** Define default styles in the template and override them for specific entries in the data file.
*   **Flexible Orientation:** Supports both horizontal and vertical timeline layouts, with optional angle overrides for specific segments.
*   **Image Embedding:** Supports local image embedding within comment blocks for self-contained SVG/PNG/JPG output.
*   **Text Wrapping & Basic Markdown:** Handles basic text formatting within comment blocks.

## Requirements

*   **Go:** Version 1.18 or later recommended.
*   **Chrome/Chromium:** Required *only* for generating PNG/JPG output. The `chromedp` library interacts with a running Chrome/Chromium instance (usually found automatically). SVG generation does not require Chrome.

## Packages Used

This project utilizes Go modules for dependency management. Key external packages include:

*   `github.com/chromedp/chromedp`: For controlling a headless Chrome/Chromium instance to render SVGs to PNG/JPG.

Standard Go libraries used include: `encoding/json`, `os`, `path/filepath`, `fmt`, `log`, `math`, `bytes`, `strings`, `regexp`, `mime`, `encoding/base64`.

## Installation

1.  **Clone the Repository:**
    ```bash
    git clone <your-repository-url>
    cd go-timeline
    ```
2.  **Fetch Dependencies:** Ensure Go modules are tidy.
    ```bash
    go mod tidy
    ```
    (This might automatically run `go get` for required packages like `chromedp`).
3.  **Build the Executable:**
    ```bash
    go build -o timeline-generator .
    ```
    (On Windows, the output will be `timeline-generator.exe`).

## Usage

The tool is run from the command line, specifying an output file, template file, data file, and the desired output format.

**Command Syntax:**

```bash
./timeline-generator -o <output-file> <template.json> <data.json> <format>
```

**Arguments:**

*   `-o <output-file>`: (Required) Path where the generated output file will be saved (e.g., `timeline.svg`, `report.png`).
*   `<template.json>`: (Required) Path to the JSON file defining the timeline's appearance and default styles.
*   `<data.json>`: (Required) Path to the JSON file containing the specific entries for the timeline.
*   `<format>`: (Required) The desired output format. Must be one of:
    *   `svg`: Generates an SVG vector image.
    *   `png`: Generates a PNG raster image (requires Chrome/Chromium).
    *   `jpg` or `jpeg`: Generates a JPG raster image (requires Chrome/Chromium).

**Example:**

```bash
# Generate an SVG file
./timeline-generator -o my_timeline.svg examples/template.json examples/data.json svg

# Generate a PNG file
./timeline-generator -o my_timeline.png examples/template.json examples/data.json png
```

## Configuration Schema

The generator uses two main JSON files: a template file for styling and layout defaults, and a data file for the timeline content.

*(For an exhaustive list of all fields and their types, please refer to `schema_definition.md` and `models.go`)*

---

### Template File (`template.json`) Schema Overview

Defines the overall look, feel, and default styles for timeline elements.

```json
{
  "center_line": {
    "width": 12,                // Thickness of the main axis line (pixels).
    "type": "solid",            // Line style ("solid", "dashed", "dotted").
    "orientation": "horizontal",// "horizontal" or "vertical".
    "angle": null,              // Optional angle (degrees) overriding orientation (0=right, 90=up).
    "color": "#BDBDBD",         // Default color of the center line segments.
    "rounded_caps": true        // Whether line ends should be rounded.
  },
  "layout": {
    "padding": 50,              // Padding around the entire SVG content (pixels).
    "entry_spacing": 260,       // Default distance between the centers of adjacent timeline entries.
    "connector_length": 55      // Default length of connector lines from axis to elements.
  },
  "global_font": { ... },       // Optional: Default FontStyle used if not specified elsewhere. (See FontStyle below)
  "period_defaults": {          // Default styles applied to each entry unless overridden.
    "year_text": { ... },       // Default style for the period/year label. (See YearTextStyle below)
    "connector": { ... },       // Default style for connector lines and dots. (See ConnectorStyle below)
    "comment_text": { ... },    // Default style for comment blocks. (See CommentTextStyle below)
    "centerline_projection": {  // Default style for the center line segment associated with an entry.
      "color": "#BDBDBD"        // Color of the segment. If empty, uses center_line.color.
    },
    "junction_marker": { ... }  // Default style for markers at entry points on the axis. (See JunctionMarkerStyle below)
  }
}
```

**Common Style Objects:**

*   **`FontStyle`**: Used in `global_font`, `year_text.font`, `comment_text.font`, `comment_text.title_font`.
    ```json
    {
      "font_family": "Arial, Helvetica, sans-serif",
      "font_size": 12,       // Font size in pixels.
      "font_weight": "normal", // "normal", "bold", CSS weight numbers (e.g., "700").
      "font_style": "normal"   // "normal", "italic".
    }
    ```
*   **`YearTextStyle`**: Style for the period label (e.g., "2023", "Q1").
    ```json
    {
      "text_color": "#424242",
      "font": { ... },         // FontStyle object.
      "shape": "circle;r=30",  // Background shape ("circle;r=auto", "circle;r=30", "rectangle;w=50;h=25", "none"). Auto radius calculates based on text size.
      "fill_color": "#FFFFFF", // Background fill color.
      "border_color": "",      // Border color.
      "border_width": 3,       // Border thickness.
      "main_axis_offset": 0,   // Offset along the direction of the timeline axis.
      "cross_axis_offset": 0   // Offset perpendicular to the timeline axis.
    }
    ```
*   **`ConnectorStyle`**: Style for lines connecting the axis to elements.
    ```json
    {
      "color": "#BDBDBD",     // Line color. If empty, uses centerline_projection.color.
      "width": 2,            // Line thickness.
      "line_type": "solid",    // "solid", "dashed", "dotted".
      "side": "",            // Override element placement side ("top", "bottom", "left", "right" relative to axis orientation). Default alternates.
      "draw_to_period": true, // Draw connector to year element? (Default: true)
      "draw_to_comment": false,// Draw connector to comment element? (Default: true)
      "dot": { ... }           // Style for the dot where the connector meets the axis. (See DotStyle below)
    }
    ```
*   **`DotStyle`**: Style for the marker at the end of the connector line (usually on the axis or offset from it).
    ```json
    {
      "shape": "none",         // "circle", "square", "arrow", "none".
      "size": 0,             // Diameter/size of the dot.
      "color": "",           // Dot color. If empty, uses connector.color.
      "visible": false,        // Is the dot drawn?
      "offset_main": 0,      // Offset along the connector's direction (towards element).
      "offset_cross": 0,     // Offset perpendicular to the connector's direction.
      "stop_at_dot": true      // Does the connector line stop at the dot's (potentially offset) position?
    }
    ```
*   **`CommentTextStyle`**: Style for the comment blocks.
    ```json
    {
      "text_color": "#757575", // Body text color.
      "font": { ... },         // FontStyle for body text.
      "title_color": "#424242",// Title text color.
      "title_font": { ... },   // FontStyle for title text.
      "title_line": { ... },   // Style for decorative line under title. (See TitleLineStyle below)
      "shape": "rectangle",    // Background shape ("rectangle", "none").
      "fill_color": "",        // Background fill.
      "border_color": "red",
      "border_width": 1,
      "border_style": "solid", // "solid", "dashed", "dotted".
      "padding": "10 10",      // CSS-style padding ("T", "T R B L", "V H"). E.g., "10", "10 20", "5 10 5 20".
      "block_width": 130,      // Optional: Fixed width for the comment block content area.
      "text_align": "left",    // Text alignment within block ("left", "center", "right").
      "main_axis_offset": 0,   // Offset along the direction of the timeline axis.
      "cross_axis_offset": 0   // Offset perpendicular to the timeline axis.
    }
    ```
*   **`TitleLineStyle`**: Style for the decorative line under comment titles.
    ```json
    {
      "visible": true,
      "color": "",        // Line color. If empty, uses comment_text.title_color.
      "width": 2,         // Line thickness.
      "length": 30,       // Line length.
      "margin": 3         // Vertical space between title text and line, and line and body.
    }
    ```
*   **`JunctionMarkerStyle`**: Style for markers at the entry points on the main axis.
    ```json
    {
      "shape": "diamond",    // "diamond", "circle", "arrow" (currently same as diamond), "none".
      "size": 18,            // Size of the marker.
      "color": null          // Marker color. If null/empty, uses connector.color or centerline_projection.color.
    }
    ```

---

### Data File (`data.json`) Schema Overview

Defines the actual content and specific overrides for each timeline entry.

```json
{
  "entries": [ // Array of TimelineEntry objects
    {
      "period": "2017",                   // Text label for the year/period element.
      "title_text": "TITLE LINE 01",      // Optional: Title displayed in the comment block.
      "comment_text": "Description...",   // Optional: Body text for the comment block. Supports \n for newlines and [link text](url).
      "comment_image": "images/img1.png", // Optional: URL or local path to an image in the comment block. Local paths are embedded.
      "link": "http://example.com",       // Optional: URL to link the year/period element to.
      "entry_spacing_override": null,     // Optional: Override layout.entry_spacing for the space *after* this entry.
      "orientation_override": null,     // Optional: Override center_line.orientation ("horizontal" or "vertical") for placement calculations *of this entry*.
      "angle_override": null,           // Optional: Override center_line.angle (degrees) for the axis segment *leading to the next entry*.
      // --- Override Blocks ---
      "year_text_override": { ... },      // Optional: Overrides fields from period_defaults.year_text.
      "connector_override": { ... },      // Optional: Overrides fields from period_defaults.connector.
      "comment_text_override": { ... },   // Optional: Overrides fields from period_defaults.comment_text.
      "centerline_projection_override": { // Optional: Overrides fields from period_defaults.centerline_projection.
        "color": "#FFCA28"
      },
      "junction_marker_override": { ... } // Optional: Overrides fields from period_defaults.junction_marker.
    },
    // ... more entries
  ]
}
```

**Overrides:**

*   Each `*_override` block within an entry allows you to change specific style properties for that entry only.
*   The structure of an override block mirrors the corresponding default style block (e.g., `ConnectorStyleOverride` mirrors `ConnectorStyle`).
*   **Important:** Fields within override blocks are typically pointers (`*string`, `*int`, `*bool`, etc.). This allows the generator to distinguish between a value explicitly set to `0` or `false` in the override versus a field that wasn't included (and should therefore keep the default value).
*   Only include the fields you want to change in the override block. Unspecified fields will inherit from the `period_defaults` in the template file.

---

## Examples

The `examples/` directory contains sample template and data files demonstrating various features:

*   `template.json`, `data.json`: Default example showcasing various overrides.
*   `template_1.json`, `data_1.json`: Another style variation.
*   `template_2.json`, `data_2.json`: Alternative style.
*   `template_3.json`, `data_3.json`: Example demonstrating connector dot offsets and different comment styling.

**Example Command:**

```bash
./timeline-generator -o examples/timeline_default.svg examples/template.json examples/data.json svg
```

## Testing

This project includes SVG comparison tests to help prevent regressions.

1.  **Setup:**
    *   Ensure the test file (`svg_generation_test.go` or similar) is in the project root directory.
    *   Ensure the `testdata/` directory is in the project root.
    *   Populate `testdata/` with pairs of `*.tmpl.json` and `*.data.json` files.
    *   For each pair, generate the "correct" SVG output and save it as `*.expected.svg` in `testdata/`. (The test will do this automatically the first time it runs for a pair if the `.expected.svg` file is missing).
2.  **Run Tests:**
    ```bash
    go test ./... -v
    ```
    (Run from the project root directory).
3.  **Update Tests:** If you make intentional changes that alter the SVG output, delete the corresponding `.expected.svg` file and re-run the tests. It will fail but generate the new expected file. Verify the new file is correct and commit it.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

MIT