package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/xuri/excelize/v2"
)

// exportSet is one dataset ready to write: the chosen column ids, the
// (already scoped, possibly capped) rows, the TOTAL row (nil = none) and how
// many rows the dataset had before the cap.
type exportSet struct {
	ds     *reportDataset
	cols   []string
	rows   []datatype.DataMap
	totals datatype.DataMap
	total  int
}

func (s exportSet) truncated() bool { return s.total > len(s.rows) }

// exportMeta is what every file's header / notes say.
type exportMeta struct {
	lang      string
	scope     string // partner / cluster / group name, "" = everything visible
	from, to  string // YYYY-MM-DD or ""
	asAt      string // today (EAT), for point-in-time datasets
	generated time.Time
}

var exportText = map[string]map[string]string{
	"sw": {
		"title": "Ripoti ya " + brandName, "scope": "Wigo", "period": "Kipindi", "gen": "Imetengenezwa",
		"none": "Hakuna data kwa vigezo hivi.", "all": "Vikundi vyote", "any": "Muda wote", "total": "JUMLA",
		"asAt":     "Hali kufikia %s (hali ya sasa; kipindi cha tarehe hakihusiki).",
		"balances": "Salio ni kufikia %s; safu za \"(kipindi)\" ni za %s.",
		"cut":      "Safu %s za kwanza tu kati ya %s ndizo zimejumuishwa.",
		"page":     "Ukurasa",
	},
	"en": {
		"title": brandName + " Report", "scope": "Scope", "period": "Period", "gen": "Generated",
		"none": "No data for these filters.", "all": "All groups", "any": "All time", "total": "TOTAL",
		"asAt":     "As at %s (current position; the date range does not apply).",
		"balances": "Balances as at %s; \"(period)\" columns cover %s.",
		"cut":      "Only the first %s of %s rows are included.",
		"page":     "Page",
	},
}

func (m exportMeta) text(key string) string { return exportText[normLang(m.lang)][key] }

func (m exportMeta) period() string {
	if m.from == "" && m.to == "" {
		return m.text("any")
	}
	from, to := m.from, m.to
	if from == "" {
		from = "..."
	}
	if to == "" {
		to = "..."
	}
	return from + " - " + to
}

// notes are the lines printed under a dataset's title (PDF) or after its
// rows (XLSX / CSV).
func (m exportMeta) notes(s exportSet) []string {
	out := []string{}
	if s.ds.PointInTime {
		out = append(out, fmt.Sprintf(m.text("asAt"), m.asAt))
	}
	if s.ds.Balances {
		out = append(out, fmt.Sprintf(m.text("balances"), m.asAt, m.period()))
	}
	if s.truncated() {
		out = append(out, fmt.Sprintf(m.text("cut"), commaInt(int64(len(s.rows))), commaInt(int64(s.total))))
	}
	return out
}

// ---------------------------------------------------------------------------
// Cell formatting
// ---------------------------------------------------------------------------

// plainNumber: whole numbers without ".00", others with 2 decimals.
func plainNumber(n float64) string {
	if n == math.Trunc(n) && math.Abs(n) < 1e15 {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', 2, 64)
}

// groupedNumber is plainNumber with thousands separators.
func groupedNumber(n float64) string {
	s := plainNumber(n)
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	v, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return s
	}
	out := commaInt(v)
	if v == 0 && strings.HasPrefix(intPart, "-") {
		out = "-" + out
	}
	return out + frac
}

// cellText renders a value of a column of type typ for CSV (pretty=false:
// plain machine-readable numbers) or PDF (pretty=true: thousands separators
// and "%"). Dates print as EAT days; enum values are translated; free text
// is left alone. nil prints as "".
func cellText(lang, typ string, v interface{}, pretty bool) string {
	if v == nil {
		return ""
	}
	switch typ {
	case "date", "datetime":
		t := parseTimeOrZero(v)
		if t.IsZero() {
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		}
		if typ == "datetime" {
			return t.In(eatZone).Format("2006-01-02 15:04")
		}
		return t.In(eatZone).Format("2006-01-02")
	case "money", "count", "int", "rate":
		switch v.(type) {
		case float64, float32, int, int32, int64:
		default:
			return fmt.Sprint(v)
		}
		n := num(v)
		if !pretty {
			return plainNumber(n)
		}
		if typ == "rate" {
			return strconv.FormatFloat(n, 'f', -1, 64) + "%"
		}
		if typ == "int" {
			return plainNumber(n)
		}
		return groupedNumber(n)
	case "enum":
		if s, ok := v.(string); ok {
			return enumLabel(normLang(lang), s)
		}
	}
	if f, ok := v.(float64); ok {
		return plainNumber(f)
	}
	return fmt.Sprint(v)
}

// totalsLabelCol is where the TOTAL label goes: the first chosen column that
// carries no total itself ("" = none free).
func totalsLabelCol(s exportSet) string {
	for _, c := range s.cols {
		if _, has := s.totals[c]; !has {
			return c
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// CSV
// ---------------------------------------------------------------------------

func csvBytes(m exportMeta, s exportSet) []byte {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM so Excel opens UTF-8 correctly
	w := csv.NewWriter(&buf)
	head := make([]string, len(s.cols))
	for i, c := range s.cols {
		head[i] = colLabel(m.lang, c)
	}
	_ = w.Write(head)
	for _, row := range s.rows {
		rec := make([]string, len(s.cols))
		for i, c := range s.cols {
			rec[i] = cellText(m.lang, colType(s.ds, c), row[c], false)
		}
		_ = w.Write(rec)
	}
	if s.totals != nil {
		label := totalsLabelCol(s)
		rec := make([]string, len(s.cols))
		for i, c := range s.cols {
			if c == label {
				rec[i] = m.text("total")
			} else {
				rec[i] = cellText(m.lang, colType(s.ds, c), s.totals[c], false)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	for _, n := range m.notes(s) {
		buf.WriteString("# " + n + "\r\n")
	}
	return buf.Bytes()
}

func exportCSV(m exportMeta, sets []exportSet) ([]byte, string, error) {
	if len(sets) == 1 {
		return csvBytes(m, sets[0]), "csv", nil
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, s := range sets {
		w, err := zw.Create(s.ds.Key + ".csv")
		if err != nil {
			return nil, "", err
		}
		_, _ = w.Write(csvBytes(m, s))
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "zip", nil
}

// ---------------------------------------------------------------------------
// XLSX
// ---------------------------------------------------------------------------

// xlsxStyles holds one style per column type, plain and bold (totals).
type xlsxStyles struct {
	head  int
	note  int
	plain map[string]int
	bold  map[string]int
}

func newXlsxStyles(f *excelize.File) (*xlsxStyles, error) {
	st := &xlsxStyles{plain: map[string]int{}, bold: map[string]int{}}
	var err error
	if st.head, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0F6E56"}, Pattern: 1},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"},
	}); err != nil {
		return nil, err
	}
	if st.note, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Italic: true, Color: "#555555"}}); err != nil {
		return nil, err
	}
	formats := map[string]string{
		"money": "#,##0", "count": "#,##0", "int": "0", "rate": `0.0"%"`,
		"date": "yyyy-mm-dd", "datetime": "yyyy-mm-dd hh:mm", "enum": "", "text": "",
	}
	for typ, fmtCode := range formats {
		for _, bold := range []bool{false, true} {
			s := &excelize.Style{}
			if fmtCode != "" {
				code := fmtCode
				s.CustomNumFmt = &code
			}
			if bold {
				s.Font = &excelize.Font{Bold: true}
				s.Fill = excelize.Fill{Type: "pattern", Color: []string{"#E3F1EC"}, Pattern: 1}
			}
			id, err := f.NewStyle(s)
			if err != nil {
				return nil, err
			}
			if bold {
				st.bold[typ] = id
			} else {
				st.plain[typ] = id
			}
		}
	}
	return st, nil
}

// xlsxValue is the typed cell value: numbers stay numbers, dates become real
// Excel dates (EAT wall-clock), enums are translated, nil / "" stay empty.
func xlsxValue(lang, typ string, v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch typ {
	case "date", "datetime":
		t := parseTimeOrZero(v)
		if t.IsZero() {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
			return nil
		}
		t = t.In(eatZone)
		if typ == "date" {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	case "money", "count", "int", "rate":
		switch n := v.(type) {
		case float64, int, int32, int64:
			return n
		}
	}
	s := cellText(lang, typ, v, false)
	if s == "" {
		return nil
	}
	return s
}

func xlsxSheetName(title string, used map[string]bool) string {
	name := strings.TrimSpace(strings.NewReplacer(":", " ", "\\", " ", "/", " ", "?", " ", "*", " ", "[", " ", "]", " ").Replace(title))
	if name == "" {
		name = "Sheet"
	}
	if r := []rune(name); len(r) > 31 {
		name = string(r[:31])
	}
	candidate := name
	for i := 2; used[strings.ToLower(candidate)]; i++ {
		suffix := fmt.Sprintf(" %d", i)
		r := []rune(name)
		if len(r)+len(suffix) > 31 {
			r = r[:31-len(suffix)]
		}
		candidate = string(r) + suffix
	}
	used[strings.ToLower(candidate)] = true
	return candidate
}

// xlsxColWidth guesses a readable width from the header and the first rows.
func xlsxColWidth(m exportMeta, s exportSet, col string) float64 {
	typ := colType(s.ds, col)
	w := float64(len([]rune(colLabel(m.lang, col))))
	if w > 24 {
		w = 24 // long headers wrap
	}
	for i, row := range s.rows {
		if i >= 300 {
			break
		}
		if l := float64(len([]rune(cellText(m.lang, typ, row[col], true)))); l > w {
			w = l
		}
	}
	switch typ {
	case "date":
		w = math.Max(w, 11)
	case "datetime":
		w = math.Max(w, 16)
	case "money":
		w = math.Max(w, 12)
	}
	return math.Min(math.Max(w+2, 8), 45)
}

func exportXLSX(m exportMeta, sets []exportSet) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	st, err := newXlsxStyles(f)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	for i, s := range sets {
		name := xlsxSheetName(dsLabel(m.lang, s.ds), used)
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				return nil, err
			}
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		sw, err := f.NewStreamWriter(name)
		if err != nil {
			return nil, err
		}
		if err := sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
			return nil, err
		}
		for c, col := range s.cols {
			if err := sw.SetColWidth(c+1, c+1, xlsxColWidth(m, s, col)); err != nil {
				return nil, err
			}
		}
		head := make([]interface{}, len(s.cols))
		for c, col := range s.cols {
			head[c] = excelize.Cell{StyleID: st.head, Value: colLabel(m.lang, col)}
		}
		if err := sw.SetRow("A1", head, excelize.RowOpts{Height: 30}); err != nil {
			return nil, err
		}
		r := 2
		writeRow := func(row datatype.DataMap, styles map[string]int, label string) error {
			vals := make([]interface{}, len(s.cols))
			for c, col := range s.cols {
				typ := colType(s.ds, col)
				var v interface{}
				if col == label && label != "" {
					v = m.text("total")
				} else {
					v = xlsxValue(m.lang, typ, row[col])
				}
				if v == nil {
					if styles[typ] != st.plain[typ] { // keep the totals row shaded across
						vals[c] = excelize.Cell{StyleID: styles[typ], Value: ""}
					}
					continue
				}
				vals[c] = excelize.Cell{StyleID: styles[typ], Value: v}
			}
			cell, _ := excelize.CoordinatesToCellName(1, r)
			r++
			return sw.SetRow(cell, vals)
		}
		for _, row := range s.rows {
			if err := writeRow(row, st.plain, ""); err != nil {
				return nil, err
			}
		}
		if len(s.rows) == 0 {
			cell, _ := excelize.CoordinatesToCellName(1, r)
			r++
			if err := sw.SetRow(cell, []interface{}{excelize.Cell{StyleID: st.note, Value: m.text("none")}}); err != nil {
				return nil, err
			}
		}
		if s.totals != nil && len(s.rows) > 0 {
			if err := writeRow(s.totals, st.bold, totalsLabelCol(s)); err != nil {
				return nil, err
			}
		}
		if notes := m.notes(s); len(notes) > 0 {
			r++ // one blank row before the notes
			for _, n := range notes {
				cell, _ := excelize.CoordinatesToCellName(1, r)
				r++
				if err := sw.SetRow(cell, []interface{}{excelize.Cell{StyleID: st.note, Value: n}}); err != nil {
					return nil, err
				}
			}
		}
		if err := sw.Flush(); err != nil {
			return nil, err
		}
		// Printed page header: dataset, scope/period and the "as at" date.
		scope := m.scope
		if scope == "" {
			scope = m.text("all")
		}
		right := m.text("period") + ": " + m.period()
		if s.ds.PointInTime || s.ds.Balances {
			right = fmt.Sprintf(m.text("asAt"), m.asAt)
		}
		esc := strings.NewReplacer("&", "&&")
		_ = f.SetHeaderFooter(name, &excelize.HeaderFooterOptions{
			OddHeader: "&L" + esc.Replace(brandName+" - "+dsLabel(m.lang, s.ds)+" - "+scope) + "&R" + esc.Replace(right),
			OddFooter: "&C" + esc.Replace(m.text("page")) + " &P / &N",
		})
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// PDF
// ---------------------------------------------------------------------------

// pdfWideText columns get the most room (names and other free text).
var pdfWideText = map[string]bool{
	"Name": true, "Group": true, "Member": true, "Borrower": true, "Title": true, "Reason": true, "Lender": true,
	"Programme": true, "Description": true, "Partner": true, "Cluster": true,
}

// pdfColWeight is a column's share of the page width.
func pdfColWeight(ds *reportDataset, col string) float64 {
	switch colType(ds, col) {
	case "money":
		return 1.45
	case "count":
		return 0.95
	case "int":
		return 0.85
	case "rate":
		return 0.95
	case "date":
		return 1.3
	case "datetime":
		return 1.75
	case "enum":
		if col == "Activity" {
			return 2.0
		}
		return 1.35
	}
	if pdfWideText[col] {
		return 2.4
	}
	return 1.35
}

func pdfFontSize(cols int) float64 {
	switch {
	case cols <= 9:
		return 9
	case cols <= 12:
		return 8
	case cols <= 16:
		return 7
	default:
		return 6.2
	}
}

// pdfLines wraps s (already cp1252) into at most max lines of width w; a
// cell that still does not fit ends in "..." at a word boundary.
func pdfLines(p *fpdf.Fpdf, s string, w float64, max int) []string {
	if s == "" {
		return []string{""}
	}
	raw := p.SplitLines([]byte(s), w)
	lines := make([]string, 0, len(raw))
	for _, l := range raw {
		lines = append(lines, string(l))
	}
	if len(lines) <= max {
		return lines
	}
	lines = lines[:max]
	last := strings.TrimRight(lines[max-1], " ")
	avail := w - 2*p.GetCellMargin()
	for p.GetStringWidth(last+"...") > avail && last != "" {
		if i := strings.LastIndexByte(last, ' '); i > 0 {
			last = strings.TrimRight(last[:i], " ")
		} else {
			last = last[:len(last)-1]
		}
	}
	lines[max-1] = last + "..."
	return lines
}

// pdfLayout picks the font size and column widths for one table.
func pdfLayout(p *fpdf.Fpdf, tr func(string) string, m exportMeta, s exportSet, usable float64) ([]float64, float64) {
	n := len(s.cols)
	weights := make([]float64, n)
	var wsum float64
	for i, c := range s.cols {
		weights[i] = pdfColWeight(s.ds, c)
		wsum += weights[i]
	}
	longestWord := func(text string) float64 {
		best := 0.0
		for _, w := range strings.Fields(tr(text)) {
			if l := p.GetStringWidth(w); l > best {
				best = l
			}
		}
		return best
	}
	sample := s.rows
	if len(sample) > 500 {
		sample = sample[:500]
	}
	for fs := pdfFontSize(n); ; fs -= 0.5 {
		mins := make([]float64, n)
		var minSum float64
		for i, c := range s.cols {
			p.SetFont("Helvetica", "B", fs)
			mins[i] = longestWord(colLabel(m.lang, c))
			if s.totals != nil {
				if l := longestWord(cellText(m.lang, colType(s.ds, c), s.totals[c], true)); l > mins[i] {
					mins[i] = l
				}
			}
			p.SetFont("Helvetica", "", fs)
			for _, row := range sample {
				if l := longestWord(cellText(m.lang, colType(s.ds, c), row[c], true)); l > mins[i] {
					mins[i] = l
				}
			}
			mins[i] = math.Min(mins[i]+2*p.GetCellMargin()+0.8, usable*0.35)
			minSum += mins[i]
		}
		if minSum <= usable || fs <= 5 {
			widths := make([]float64, n)
			if minSum > usable { // still too wide at the smallest font: squeeze
				for i := range widths {
					widths[i] = mins[i] * usable / minSum
				}
				return widths, fs
			}
			for i := range widths {
				widths[i] = mins[i] + (usable-minSum)*weights[i]/wsum
			}
			return widths, fs
		}
	}
}

func exportPDF(m exportMeta, sets []exportSet) ([]byte, error) {
	const (
		pageH   = 210.0
		bottom  = 14.0
		usable  = 277.0 // A4 landscape 297 - 2*10 margins
		marginL = 10.0
	)
	p := fpdf.New("L", "mm", "A4", "")
	// Core fonts are cp1252, which covers every Swahili/English character we
	// print; the translator maps our UTF-8 strings (e.g. the "—" placeholder).
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.SetMargins(marginL, 12, 10)
	p.SetAutoPageBreak(false, bottom)
	p.SetFooterFunc(func() {
		p.SetY(-10)
		p.SetFont("Helvetica", "", 8)
		p.SetTextColor(120, 120, 120)
		p.CellFormat(0, 6, tr(fmt.Sprintf("%s  |  %s %d", brandName, m.text("page"), p.PageNo())), "", 0, "C", false, 0, "")
	})
	p.AddPage()
	room := func(h float64) {
		if p.GetY()+h > pageH-bottom {
			p.AddPage()
		}
	}

	scope := m.scope
	if scope == "" {
		scope = m.text("all")
	}
	p.SetFont("Helvetica", "B", 18)
	p.SetTextColor(15, 110, 86)
	p.CellFormat(0, 10, tr(m.text("title")), "", 1, "L", false, 0, "")
	p.SetFont("Helvetica", "", 10)
	p.SetTextColor(70, 70, 70)
	p.CellFormat(0, 6, tr(fmt.Sprintf("%s: %s    %s: %s    %s: %s EAT", m.text("scope"), scope, m.text("period"), m.period(),
		m.text("gen"), m.generated.In(eatZone).Format("2006-01-02 15:04"))), "", 1, "L", false, 0, "")
	p.Ln(4)

	for _, s := range sets {
		room(30)
		p.SetFont("Helvetica", "B", 13)
		p.SetTextColor(15, 110, 86)
		p.CellFormat(0, 8, tr(fmt.Sprintf("%s (%s)", dsLabel(m.lang, s.ds), commaInt(int64(s.total)))), "", 1, "L", false, 0, "")
		p.SetFont("Helvetica", "I", 8.5)
		p.SetTextColor(90, 90, 90)
		for _, n := range m.notes(s) {
			p.CellFormat(0, 5, tr(n), "", 1, "L", false, 0, "")
		}

		if len(s.rows) == 0 {
			p.SetFont("Helvetica", "I", 10)
			p.SetTextColor(120, 120, 120)
			p.CellFormat(0, 7, tr(m.text("none")), "", 1, "L", false, 0, "")
			p.Ln(3)
			continue
		}

		// Column widths: every column gets its longest word (so names and
		// numbers never break mid-word), the rest is shared by per-type weight;
		// the font shrinks until the minimums fit the page.
		widths, fs := pdfLayout(p, tr, m, s, usable)
		lineH := fs * 0.42
		aligns := make([]string, len(s.cols))
		for i, c := range s.cols {
			aligns[i] = "L"
			switch colType(s.ds, c) {
			case "money", "count", "int", "rate":
				aligns[i] = "R"
			}
		}

		// drawRow draws one table row of wrapped cells; style "B" = bold.
		drawRow := func(cells []string, style string, fill [3]int, textColor [3]int, maxLines int, alignAll string) {
			p.SetFont("Helvetica", style, fs)
			lines := make([][]string, len(cells))
			n := 1
			for i, c := range cells {
				lines[i] = pdfLines(p, tr(c), widths[i], maxLines)
				if len(lines[i]) > n {
					n = len(lines[i])
				}
			}
			h := float64(n)*lineH + 1.6
			x, y := marginL, p.GetY()
			p.SetFillColor(fill[0], fill[1], fill[2])
			p.SetDrawColor(200, 210, 206)
			p.SetTextColor(textColor[0], textColor[1], textColor[2])
			for i := range cells {
				p.Rect(x, y, widths[i], h, "FD")
				align := aligns[i]
				if alignAll != "" {
					align = alignAll
				}
				for j, l := range lines[i] {
					p.SetXY(x, y+0.8+float64(j)*lineH)
					p.CellFormat(widths[i], lineH, l, "", 0, align, false, 0, "")
				}
				x += widths[i]
			}
			p.SetXY(marginL, y+h)
		}
		headerCells := make([]string, len(s.cols))
		for i, c := range s.cols {
			headerCells[i] = colLabel(m.lang, c)
		}
		header := func() {
			drawRow(headerCells, "B", [3]int{15, 110, 86}, [3]int{255, 255, 255}, 5, "L")
		}
		rowHeight := func(cells []string, style string) float64 {
			p.SetFont("Helvetica", style, fs)
			n := 1
			for i, c := range cells {
				if l := len(pdfLines(p, tr(c), widths[i], 3)); l > n {
					n = l
				}
			}
			return float64(n)*lineH + 1.6
		}
		room(rowHeight(headerCells, "B") + 3*lineH + 4)
		header()
		for i, row := range s.rows {
			cells := make([]string, len(s.cols))
			for j, c := range s.cols {
				cells[j] = cellText(m.lang, colType(s.ds, c), row[c], true)
			}
			if p.GetY()+rowHeight(cells, "") > pageH-bottom {
				p.AddPage()
				header()
			}
			fill := [3]int{255, 255, 255}
			if i%2 == 1 {
				fill = [3]int{240, 246, 244}
			}
			drawRow(cells, "", fill, [3]int{30, 30, 30}, 3, "")
		}
		if s.totals != nil {
			label := totalsLabelCol(s)
			cells := make([]string, len(s.cols))
			for j, c := range s.cols {
				if c == label {
					cells[j] = m.text("total")
				} else {
					cells[j] = cellText(m.lang, colType(s.ds, c), s.totals[c], true)
				}
			}
			if p.GetY()+rowHeight(cells, "B") > pageH-bottom {
				p.AddPage()
				header()
			}
			drawRow(cells, "B", [3]int{227, 241, 236}, [3]int{15, 60, 45}, 3, "")
		}
		p.Ln(6)
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
