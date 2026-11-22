package printout

// Shape alur di Excel (bagian B): kotak, belah ketupat, kotak NG, garis
// berpanah, dan konektor, untuk kolom Process Flow CP dan diagram PFD.
//
// excelize belum bisa memberi kepala panah pada garis dan belum bisa membuat
// teks shape rata tengah. Keduanya ditambahkan sesudah berkas ditulis
// (fixDrawings): shape diberi nama berawalan "arrow-end", "arrow-start", atau
// "c-", lalu XML drawing-nya dilengkapi.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// drawer menggambar shape di satu sheet. abs mengubah posisi lokal menjadi
// posisi mutlak di lembar, dalam piksel Excel. Posisi dan ukuran ditulis di
// nama shape (lihat place), lalu fixDrawings mengganti jangkar sel buatan
// excelize dengan jangkar mutlak. Dengan begitu bentuk tidak memanjang walau
// excelize dan Excel menghitung lebar kolom dan tinggi baris berbeda.
type drawer struct {
	f    *excelize.File
	name string
	abs  func(x, y float64) (float64, float64)
	n    int // nomor urut nama shape
}

// place memberi nama shape beserta posisi mutlaknya dalam EMU:
// awalan+nomor@x,y,lebar,tinggi.
func (d *drawer) place(prefix string, x, y, w, h float64) string {
	ax, ay := d.abs(x, y)
	d.n++
	emu := func(v float64) int { return int(math.Round(v * 9525)) }
	return fmt.Sprintf("%s%d@%d,%d,%d,%d", prefix, d.n, emu(ax), emu(ay), emu(math.Max(1, w)), emu(math.Max(1, h)))
}

var (
	black   = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"000000"}}
	white   = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFFFF"}}
	outline = excelize.LineOptions{Width: 0.75, Fill: black}
)

// excelShapes: jenis shape Excel untuk setiap simbol alur.
var excelShapes = map[string]string{"decision": "flowChartDecision", "merge": "flowChartMerge", "connector": "flowChartConnector",
	"alternate_process": "flowChartPreparation", "document": "flowChartDocument"}

func excelShape(symbol string) string {
	if t := excelShapes[symbol]; t != "" {
		return t
	}
	return "flowChartProcess"
}

// shape menggambar satu shape dengan teks rata tengah, satu paragraf per baris.
func (d *drawer) shape(typ string, x, y, w, h float64, lines []string, size float64) error {
	s := &excelize.Shape{Cell: "A1", Type: typ, Width: uint(math.Max(1, math.Round(w))), Height: uint(math.Max(1, math.Round(h))), Fill: white, Line: outline,
		Format: excelize.GraphicOptions{Name: d.place("c-", x, y, w, h)}}
	for _, l := range lines {
		s.Paragraph = append(s.Paragraph, excelize.RichTextRun{Text: l, Font: &excelize.Font{Bold: true, Size: size, Color: "000000", Family: "Arial"}})
	}
	return d.f.AddShape(d.name, s)
}

// line menggambar garis mendatar atau tegak dari (x1, y1) ke (x2, y2). Bila
// arrow, ujung (x2, y2) diberi kepala panah.
func (d *drawer) line(x1, y1, x2, y2 float64, arrow bool) error {
	prefix := "line-"
	if arrow {
		prefix = "arrow-end-"
		if x2 < x1 || y2 < y1 {
			prefix = "arrow-start-"
		}
	}
	left, top := math.Min(x1, x2), math.Min(y1, y2)
	w, h := math.Abs(x2-x1), math.Abs(y2-y1)
	// Garis mendatar atau tegak: tebal nol di arah lainnya.
	name := d.place(prefix, left, top, w, h)
	if w < 1 {
		name = name[:strings.LastIndex(name, "@")] + "@" + zeroSize(name, 2)
	}
	if h < 1 {
		name = name[:strings.LastIndex(name, "@")] + "@" + zeroSize(name, 3)
	}
	s := &excelize.Shape{Cell: "A1", Type: "line", Width: uint(math.Max(1, math.Round(w))), Height: uint(math.Max(1, math.Round(h))), Line: outline,
		Format: excelize.GraphicOptions{Name: name}}
	return d.f.AddShape(d.name, s)
}

// text menulis label tanpa garis tepi (NG, OK).
func (d *drawer) text(x, y, w, h float64, label string, size float64) error {
	s := &excelize.Shape{Cell: "A1", Type: "rect", Width: uint(math.Max(1, math.Round(w))), Height: uint(math.Max(1, math.Round(h))),
		Line:      excelize.LineOptions{Width: 0},
		Format:    excelize.GraphicOptions{Name: d.place("t-", x, y, w, h)},
		Paragraph: []excelize.RichTextRun{{Text: label, Font: &excelize.Font{Size: size, Color: "000000", Family: "Arial"}}}}
	return d.f.AddShape(d.name, s)
}

// zeroSize menulis ulang angka posisi di nama shape dengan ukuran ke-i
// (2 lebar, 3 tinggi) menjadi nol.
func zeroSize(name string, i int) string {
	parts := strings.Split(name[strings.LastIndex(name, "@")+1:], ",")
	parts[i] = "0"
	return strings.Join(parts, ",")
}

// ngBox menggambar kotak tujuan NG: predefined process untuk SCRAP dan NCP,
// kotak proses untuk REWORK.
func (d *drawer) ngBox(x, y, w, h float64, ng string, size float64) error {
	typ := "flowChartPredefinedProcess"
	if !predefined(ng) {
		typ = "flowChartProcess"
	}
	return d.shape(typ, x, y, w, h, []string{ngLabel(ng)}, size)
}

var (
	anchorRe   = regexp.MustCompile(`(?s)<xdr:twoCellAnchor[^>]*>\s*<xdr:from>.*?</xdr:to>(.*?)</xdr:twoCellAnchor>`)
	placeRe    = regexp.MustCompile(`name="[^"]*@(\d+),(\d+),(\d+),(\d+)"`)
	spRe       = regexp.MustCompile(`(?s)<xdr:sp\b.*?</xdr:sp>`)
	nameRe     = regexp.MustCompile(`<xdr:cNvPr [^>]*name="([^"]*)"`)
	lnCloseRe  = regexp.MustCompile(`</a:ln>`)
	lnEmptyRe  = regexp.MustCompile(`<a:ln( [^>]*)?/>`)
	bodyTopRe  = regexp.MustCompile(`anchor="t"`)
	paraOpenRe = regexp.MustCompile(`<a:p>`)
	noLineRe   = regexp.MustCompile(`<a:ln( [^>]*)?>.*?</a:ln>|<a:ln( [^>]*)?/>`)
)

// fixDrawingXML melengkapi XML drawing: kepala panah, teks rata tengah, dan
// label tanpa garis tepi.
func fixDrawingXML(x string) string {
	// Jangkar mutlak untuk shape yang posisinya tertulis di namanya.
	x = anchorRe.ReplaceAllStringFunc(x, func(a string) string {
		m := anchorRe.FindStringSubmatch(a)
		p := placeRe.FindStringSubmatch(m[1])
		if p == nil {
			return a
		}
		return `<xdr:absoluteAnchor><xdr:pos x="` + p[1] + `" y="` + p[2] + `"/><xdr:ext cx="` + p[3] + `" cy="` + p[4] + `"/>` + m[1] + `</xdr:absoluteAnchor>`
	})
	return spRe.ReplaceAllStringFunc(x, func(sp string) string {
		m := nameRe.FindStringSubmatch(sp)
		if m == nil {
			return sp
		}
		end := ""
		switch {
		case strings.HasPrefix(m[1], "arrow-end-"):
			end = `<a:tailEnd type="triangle"/>`
		case strings.HasPrefix(m[1], "arrow-start-"):
			end = `<a:headEnd type="triangle"/>`
		case strings.HasPrefix(m[1], "c-"):
			sp = bodyTopRe.ReplaceAllString(sp, `anchor="ctr"`)
			sp = strings.Replace(sp, `wrap="none"`, `wrap="square" lIns="0" rIns="0" tIns="0" bIns="0"`, 1)
			return paraOpenRe.ReplaceAllString(sp, `<a:p><a:pPr algn="ctr"/>`)
		case strings.HasPrefix(m[1], "t-"):
			sp = bodyTopRe.ReplaceAllString(sp, `anchor="ctr"`)
			sp = noLineRe.ReplaceAllString(sp, `<a:ln><a:noFill/></a:ln>`)
			return strings.Replace(sp, `<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill>`, `<a:noFill/>`, 1)
		}
		if end == "" {
			return sp
		}
		if lnCloseRe.MatchString(sp) {
			return lnCloseRe.ReplaceAllString(sp, end+`</a:ln>`)
		}
		return lnEmptyRe.ReplaceAllStringFunc(sp, func(ln string) string {
			return strings.TrimSuffix(ln, "/>") + ">" + end + "</a:ln>"
		})
	})
}

// fixDrawings menjalankan fixDrawingXML pada setiap drawing di berkas Excel.
func fixDrawings(body []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(f.Name, "xl/drawings/drawing") && strings.HasSuffix(f.Name, ".xml") {
			data = []byte(fixDrawingXML(string(data)))
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: f.Modified})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
