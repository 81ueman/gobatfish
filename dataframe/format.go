package dataframe

import (
	"strings"
)

// String renders the frame the way pandas.DataFrame.to_string does: a left
// index column followed by aligned columns.
func (d *DataFrame) String() string {
	if len(d.columns) == 0 {
		return ""
	}
	rows, cols := d.nrows, len(d.columns)

	// Determine per-column alignment and width.
	widths := make([]int, cols)
	numeric := make([]bool, cols)
	for c, name := range d.columns {
		numeric[c] = true
		widths[c] = len(name)
		for i := 0; i < rows; i++ {
			v := d.cols[name].At(i)
			if v != nil && !isNumber(v) {
				numeric[c] = false
			}
			if l := len(renderValue(v)); l > widths[c] {
				widths[c] = l
			}
		}
	}

	indexWidth := len("0")
	if rows > 0 {
		indexWidth = len(renderValue(rows - 1))
	}

	var b strings.Builder
	// Header.
	b.WriteString(strings.Repeat(" ", indexWidth))
	for c, name := range d.columns {
		b.WriteString(" ")
		b.WriteString(pad(name, widths[c], numeric[c]))
	}
	for i := 0; i < rows; i++ {
		b.WriteString("\n")
		b.WriteString(pad(renderValue(i), indexWidth, true))
		for c, name := range d.columns {
			b.WriteString(" ")
			b.WriteString(pad(renderValue(d.cols[name].At(i)), widths[c], numeric[c]))
		}
	}
	return b.String()
}

func pad(s string, width int, rightAlign bool) string {
	if len(s) >= width {
		return s
	}
	fill := strings.Repeat(" ", width-len(s))
	if rightAlign {
		return fill + s
	}
	return s + fill
}

// HTML renders the frame as an HTML table, mirroring pandas DataFrame
// HTML representation.
func (d *DataFrame) HTML() string {
	var b strings.Builder
	b.WriteString(`<table border="1" class="dataframe">`)
	b.WriteString("<thead><tr><th></th>")
	for _, name := range d.columns {
		b.WriteString("<th>")
		b.WriteString(escapeHTML(name))
		b.WriteString("</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for i := 0; i < d.nrows; i++ {
		b.WriteString(`<tr><th>`)
		b.WriteString(escapeHTML(renderValue(i)))
		b.WriteString("</th>")
		for _, name := range d.columns {
			b.WriteString("<td>")
			b.WriteString(escapeHTML(renderValue(d.cols[name].At(i))))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;", "'", "&#39;")
	return r.Replace(s)
}
