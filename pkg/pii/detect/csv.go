package detect

import "strings"

// CSVLayout maps byte offsets to (row, column) for delimiter-separated text,
// so a value can be associated with its column header, which is what the
// product does for CSV/Excel files.
type CSVLayout struct {
	Delim  byte
	Header []string
	Rows   []Row
}

// Row is one line of cells with byte offsets into the document.
type Row struct {
	Start, End int
	Cells      []Cell
}

// Cell is one field.
type Cell struct{ Start, End int }

// ParseCSVLayout splits text into rows and cells, honouring double quotes.
func ParseCSVLayout(text string) *CSVLayout {
	l := &CSVLayout{Delim: detectDelim(text)}
	pos := 0
	for pos <= len(text) {
		nl := strings.IndexByte(text[pos:], '\n')
		lineEnd := len(text)
		next := len(text) + 1
		if nl >= 0 {
			lineEnd = pos + nl
			next = lineEnd + 1
		}
		row := Row{Start: pos, End: lineEnd}
		cellStart := pos
		inQuote := false
		for i := pos; i < lineEnd; i++ {
			c := text[i]
			if c == '"' {
				inQuote = !inQuote
				continue
			}
			if c == l.Delim && !inQuote {
				row.Cells = append(row.Cells, Cell{cellStart, i})
				cellStart = i + 1
			}
		}
		row.Cells = append(row.Cells, Cell{cellStart, lineEnd})
		if lineEnd > pos { // skip empty lines
			l.Rows = append(l.Rows, row)
		}
		pos = next
	}
	if len(l.Rows) > 0 {
		for _, c := range l.Rows[0].Cells {
			l.Header = append(l.Header, strings.Trim(strings.TrimSpace(text[c.Start:c.End]), `"`))
		}
	}
	return l
}

func detectDelim(text string) byte {
	first := text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		first = text[:i]
	}
	best, bestN := byte(','), -1
	for _, d := range []byte{',', '\t', ';', '|'} {
		if n := strings.Count(first, string(d)); n > bestN {
			best, bestN = d, n
		}
	}
	return best
}

// ColumnAt returns the column index and header of the cell containing offset.
func (l *CSVLayout) ColumnAt(off int) (int, string, bool) {
	for ri, r := range l.Rows {
		if off < r.Start || off >= r.End {
			continue
		}
		if ri == 0 {
			return -1, "", false // the header row itself
		}
		for ci, c := range r.Cells {
			if off >= c.Start && off < c.End {
				hdr := ""
				if ci < len(l.Header) {
					hdr = l.Header[ci]
				}
				return ci, hdr, true
			}
		}
	}
	return -1, "", false
}
