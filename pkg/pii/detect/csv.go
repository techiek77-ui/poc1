package detect

import (
	"sort"
	"strings"
)

// CSVLayout maps byte offsets to (row, column) for delimiter-separated text,
// so a value can be associated with its column header, which is what the
// product does for CSV/Excel files.
type CSVLayout struct {
	Delim  byte
	Header []string
	Rows   []Row
}

// Row is one record with byte offsets into the document.
type Row struct {
	Start, End int
	Cells      []Cell
}

// Cell is one field.
type Cell struct{ Start, End int }

// ParseCSVLayout splits text into rows and cells. A double quote only opens a
// field at the start of a cell and only closes it right before a delimiter,
// line end or end of text; "" inside a quoted field is an escaped quote. A
// stray quote in the middle of a field (5'11" tall) is therefore ordinary
// text and cannot swallow the rest of the file. Quoted fields may span lines.
func ParseCSVLayout(text string) *CSVLayout {
	l := &CSVLayout{Delim: detectDelim(text)}
	row := Row{Start: 0}
	cellStart := 0
	inQuote := false
	endRow := func(end, next int) {
		row.Cells = append(row.Cells, Cell{cellStart, end})
		row.End = end
		if end > row.Start {
			l.Rows = append(l.Rows, row)
		}
		row = Row{Start: next}
		cellStart = next
	}
	atCellStart := func(i int) bool {
		for j := cellStart; j < i; j++ {
			if text[j] != ' ' {
				return false
			}
		}
		return true
	}
	closesCell := func(i int) bool {
		j := i + 1
		for j < len(text) && text[j] == ' ' {
			j++
		}
		return j >= len(text) || text[j] == l.Delim || text[j] == '\n' || text[j] == '\r'
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c == '"' && !inQuote && atCellStart(i):
			inQuote = true
		case c == '"' && inQuote && i+1 < len(text) && text[i+1] == '"':
			i++ // escaped quote
		case c == '"' && inQuote && closesCell(i):
			inQuote = false
		case c == l.Delim && !inQuote:
			row.Cells = append(row.Cells, Cell{cellStart, i})
			cellStart = i + 1
		case c == '\n' && !inQuote:
			end := i
			if end > row.Start && text[end-1] == '\r' {
				end--
			}
			endRow(end, i+1)
		}
	}
	endRow(len(text), len(text)+1)
	if len(l.Rows) > 0 {
		for _, c := range l.Rows[0].Cells {
			l.Header = append(l.Header, strings.Trim(strings.TrimSpace(text[c.Start:c.End]), `"`))
		}
	}
	return l
}

// detectDelim counts candidate delimiters outside quotes over the first lines
// and prefers the one whose count is the same on every line.
func detectDelim(text string) byte {
	cands := []byte{',', '\t', ';', '|'}
	counts := map[byte][]int{}
	lines := 0
	cur := map[byte]int{}
	inQuote := false
	for i := 0; i < len(text) && lines < 5; i++ {
		c := text[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == '\n' && !inQuote:
			for _, d := range cands {
				counts[d] = append(counts[d], cur[d])
			}
			cur = map[byte]int{}
			lines++
		case !inQuote:
			for _, d := range cands {
				if c == d {
					cur[d]++
				}
			}
		}
	}
	if lines == 0 {
		for _, d := range cands {
			counts[d] = append(counts[d], cur[d])
		}
	}
	best, bestScore := byte(','), -1
	for _, d := range cands {
		cs := counts[d]
		if len(cs) == 0 || cs[0] == 0 {
			continue
		}
		consistent := true
		for _, n := range cs {
			if n != cs[0] {
				consistent = false
			}
		}
		score := cs[0]
		if consistent {
			score += 1000
		}
		if score > bestScore {
			best, bestScore = d, score
		}
	}
	return best
}

// ColumnAt returns the column index and header of the cell containing offset.
func (l *CSVLayout) ColumnAt(off int) (int, string, bool) {
	ri := sort.Search(len(l.Rows), func(i int) bool { return l.Rows[i].End > off })
	if ri >= len(l.Rows) || off < l.Rows[ri].Start {
		return -1, "", false
	}
	if ri == 0 {
		return -1, "", false // the header row itself
	}
	cells := l.Rows[ri].Cells
	ci := sort.Search(len(cells), func(i int) bool { return cells[i].End > off })
	if ci >= len(cells) || off < cells[ci].Start {
		return -1, "", false
	}
	hdr := ""
	if ci < len(l.Header) {
		hdr = l.Header[ci]
	}
	return ci, hdr, true
}
