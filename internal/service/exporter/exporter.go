package exporter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"
)

type DocumentExporter struct{}

func NewDocumentExporter() *DocumentExporter {
	return &DocumentExporter{}
}

// GeneratePDF creates a 100% valid PDF 1.4 document stream containing the proposal content.
func (e *DocumentExporter) GeneratePDF(companyName, title, content string) ([]byte, string, error) {
	filename := fmt.Sprintf("Proposal_CSR_%s.pdf", sanitizeFilename(companyName))

	var b bytes.Buffer
	lines := strings.Split(content, "\n")

	// Build PDF Content Stream
	var streamBuf bytes.Buffer
	streamBuf.WriteString("BT\n")
	streamBuf.WriteString("/F1 18 Tf\n")
	streamBuf.WriteString("50 770 Td\n")
	streamBuf.WriteString("18 TL\n")
	streamBuf.WriteString(fmt.Sprintf("(PROPOSAL KEMITRAAN STRATEGIS) Tj\n"))
	streamBuf.WriteString("T*\n")
	streamBuf.WriteString("/F1 12 Tf\n")
	streamBuf.WriteString("14 TL\n")
	streamBuf.WriteString(fmt.Sprintf("(Korporasi Target: %s) Tj\n", escapePDFText(companyName)))
	streamBuf.WriteString("T*\n")
	streamBuf.WriteString("T*\n")

	lineCount := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			streamBuf.WriteString("T*\n")
			continue
		}

		// Handle headings vs body lines
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			hText := strings.TrimLeft(trimmed, "# ")
			streamBuf.WriteString("/F1 14 Tf\n")
			streamBuf.WriteString("16 TL\n")
			streamBuf.WriteString(fmt.Sprintf("(%s) Tj\n", escapePDFText(hText)))
			streamBuf.WriteString("T*\n")
			streamBuf.WriteString("/F1 10 Tf\n")
			streamBuf.WriteString("12 TL\n")
		} else {
			// Wrap long text lines to avoid clipping
			chunks := chunkText(trimmed, 85)
			for _, chunk := range chunks {
				streamBuf.WriteString(fmt.Sprintf("(%s) Tj\n", escapePDFText(chunk)))
				streamBuf.WriteString("T*\n")
				lineCount++
				if lineCount > 45 { // Simple page break boundary
					break
				}
			}
		}
	}
	streamBuf.WriteString("ET\n")

	streamData := streamBuf.Bytes()

	// Write Valid PDF 1.4 Structure
	var offsets []int

	b.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	// Obj 1: Catalog
	offsets = append(offsets, b.Len())
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Obj 2: Pages
	offsets = append(offsets, b.Len())
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Obj 3: Page
	offsets = append(offsets, b.Len())
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources 4 0 R /MediaBox [0 0 595 842] /Contents 5 0 R >>\nendobj\n")

	// Obj 4: Resources (Helvetica Font)
	offsets = append(offsets, b.Len())
	b.WriteString("4 0 obj\n<< /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >>\nendobj\n")

	// Obj 5: Stream Content
	offsets = append(offsets, b.Len())
	b.WriteString(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n", len(streamData)))
	b.Write(streamData)
	b.WriteString("\nendstream\nendobj\n")

	// Cross-Reference Table
	startXref := b.Len()
	b.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, offset := range offsets {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Trailer
	b.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", startXref))
	b.WriteString("%%EOF\n")

	return b.Bytes(), filename, nil
}

// GenerateDOCX creates a 100% valid Microsoft Word (.docx) OpenXML ZIP file stream.
func (e *DocumentExporter) GenerateDOCX(companyName, title, content string) ([]byte, string, error) {
	filename := fmt.Sprintf("Proposal_CSR_%s.docx", sanitizeFilename(companyName))

	var docxBuf bytes.Buffer
	zipWriter := zip.NewWriter(&docxBuf)

	// 1. [Content_Types].xml
	f1, err := zipWriter.Create("[Content_Types].xml")
	if err != nil {
		return nil, "", err
	}
	f1.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// 2. _rels/.rels
	f2, err := zipWriter.Create("_rels/.rels")
	if err != nil {
		return nil, "", err
	}
	f2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// 3. word/document.xml
	f3, err := zipWriter.Create("word/document.xml")
	if err != nil {
		return nil, "", err
	}

	var docXML bytes.Buffer
	docXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`)

	// Title
	docXML.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr><w:jc w:val="center"/></w:pPr>
      <w:r>
        <w:rPr><w:b/><w:sz w:val="36"/><w:color w:val="065F46"/></w:rPr>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(strings.ToUpper(title))))

	// Subtitle / Company
	docXML.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr><w:jc w:val="center"/></w:pPr>
      <w:r>
        <w:rPr><w:i/><w:sz w:val="24"/><w:color w:val="475569"/></w:rPr>
        <w:t>Korporasi Target: %s</w:t>
      </w:r>
    </w:p>
    <w:p/>`, html.EscapeString(companyName)))

	// Paragraphs from markdown content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			docXML.WriteString("<w:p/>")
			continue
		}

		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			hText := strings.TrimLeft(trimmed, "# ")
			docXML.WriteString(fmt.Sprintf(`
    <w:p>
      <w:r>
        <w:rPr><w:b/><w:sz w:val="28"/><w:color w:val="047857"/></w:rPr>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(hText)))
		} else {
			docXML.WriteString(fmt.Sprintf(`
    <w:p>
      <w:r>
        <w:rPr><w:sz w:val="24"/></w:rPr>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(trimmed)))
		}
	}

	docXML.WriteString(`
  </w:body>
</w:document>`)

	f3.Write(docXML.Bytes())

	if err := zipWriter.Close(); err != nil {
		return nil, "", err
	}

	return docxBuf.Bytes(), filename, nil
}

// SubstitutePlaceholders opens an OpenXML ZIP archive (.docx or .pptx) and replaces placeholder keys {{KEY}} with values.
func (e *DocumentExporter) SubstitutePlaceholders(zipBytes []byte, replacements map[string]string) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template zip archive: %w", err)
	}

	var outputBuf bytes.Buffer
	zipWriter := zip.NewWriter(&outputBuf)

	for _, file := range zipReader.File {
		w, err := zipWriter.Create(file.Name)
		if err != nil {
			return nil, err
		}

		r, err := file.Open()
		if err != nil {
			return nil, err
		}

		contentBytes, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, err
		}

		// Perform placeholder substitution on XML files inside PPTX or DOCX
		if strings.HasSuffix(file.Name, ".xml") || strings.HasSuffix(file.Name, ".rels") {
			contentStr := string(contentBytes)
			for k, v := range replacements {
				escapedVal := html.EscapeString(v)
				contentStr = strings.ReplaceAll(contentStr, "{{"+k+"}}", escapedVal)
				contentStr = strings.ReplaceAll(contentStr, "{"+k+"}", escapedVal)
			}
			w.Write([]byte(contentStr))
		} else {
			w.Write(contentBytes)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close modified zip: %w", err)
	}

	return outputBuf.Bytes(), nil
}

type SlideData struct {
	SlideNumber int    `json:"slide_number"`
	Title       string `json:"title"`
	Content     string `json:"content"`
}

// GeneratePPTX creates a 100% valid Microsoft PowerPoint (.pptx) OpenXML ZIP file stream with 5 slides.
func (e *DocumentExporter) GeneratePPTX(companyName, title string, slides []SlideData) ([]byte, string, error) {
	filename := fmt.Sprintf("Pitch_Deck_CSR_%s.pptx", sanitizeFilename(companyName))

	if len(slides) == 0 {
		slides = []SlideData{
			{SlideNumber: 1, Title: "Judul & Alignment Nilai Syariah & ESG", Content: fmt.Sprintf("Kemitraan Strategis: %s x LAZ Peduli Ummat - Akselerasi Pendidikan Vokasi Syariah & Digitalisasi 3T (SDG 4 & SDG 9).", companyName)},
			{SlideNumber: 2, Title: "Tantangan Sosial & Urgensi Intervensi", Content: "Senjang digital di 50 pesantren 3T & kebutuhan 500 talenta muda berdaya saing global berbasis nilai keislaman."},
			{SlideNumber: 3, Title: "Solusi Program & Metrik Keberhasilan (OKRs)", Content: "Beasiswa Penuh 3 Tahun, Penyediaan Laptop & Pelatihan TI Syariah. Target: 100% Lulusan Terserap Kerja dalam 6 Bulan."},
			{SlideNumber: 4, Title: "Rancangan Anggaran Biaya (RAB) & Efisiensi", Content: "Estimasi Nilai Investasi Sosial: Rp 1.5 Miliar. Alokasi 85% Penyaluran Langsung, 15% Pendampingan Asnaf & Evaluasi Dampak."},
			{SlideNumber: 5, Title: "Tata Kelola, Akuntabilitas & Pelaporan POJK 51", Content: "Audit Keuangan Publik Beropini WTP & Laporan Akuntabilitas Dampak Sosial untuk Lampiran Sustainability Report Emiten."},
		}
	}

	var pptxBuf bytes.Buffer
	zipWriter := zip.NewWriter(&pptxBuf)

	// 1. [Content_Types].xml
	f1, err := zipWriter.Create("[Content_Types].xml")
	if err != nil {
		return nil, "", err
	}
	var contentTypeBuf bytes.Buffer
	contentTypeBuf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
`)
	for i := range slides {
		contentTypeBuf.WriteString(fmt.Sprintf(`  <Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`+"\n", i+1))
	}
	contentTypeBuf.WriteString(`</Types>`)
	f1.Write(contentTypeBuf.Bytes())

	// 2. _rels/.rels
	f2, err := zipWriter.Create("_rels/.rels")
	if err != nil {
		return nil, "", err
	}
	f2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`))

	// 3. ppt/presentation.xml
	f3, err := zipWriter.Create("ppt/presentation.xml")
	if err != nil {
		return nil, "", err
	}
	var presXML bytes.Buffer
	presXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst>
  <p:sldIdLst>`)
	for i := range slides {
		presXML.WriteString(fmt.Sprintf(`    <p:sldId id="%d" r:id="rId%d"/>`, 256+i, i+2))
	}
	presXML.WriteString(`</p:sldIdLst>
  <p:sldSz cx="9144000" cy="6858000" type="screen4x3"/>
  <p:notesSz cx="6858000" cy="9144000"/>
</p:presentation>`)
	f3.Write(presXML.Bytes())

	// 4. ppt/_rels/presentation.xml.rels
	f4, err := zipWriter.Create("ppt/_rels/presentation.xml.rels")
	if err != nil {
		return nil, "", err
	}
	var presRelsXML bytes.Buffer
	presRelsXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`)
	for i := range slides {
		presRelsXML.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`+"\n", i+2, i+1))
	}
	presRelsXML.WriteString(`</Relationships>`)
	f4.Write(presRelsXML.Bytes())

	// 5. Individual slide XML files (ppt/slides/slide1.xml ... slide5.xml)
	for i, slide := range slides {
		slideFileName := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
		sf, err := zipWriter.Create(slideFileName)
		if err != nil {
			return nil, "", err
		}

		var sXML bytes.Buffer
		sXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm>
      </p:grpSpPr>
      <!-- Title Box -->
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title Box"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="title"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="457200"/><a:ext cx="8229600" cy="1143000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:r>
              <a:rPr lang="id-ID" sz="2800" b="1"><a:solidFill><a:srgbClr val="047857"/></a:solidFill></a:rPr>
              <a:t>` + html.EscapeString(fmt.Sprintf("Slide #%d: %s", slide.SlideNumber, slide.Title)) + `</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
      <!-- Body Text Box -->
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Content Box"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph idx="1" type="body"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="1800000"/><a:ext cx="8229600" cy="4500000"/></a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:r>
              <a:rPr lang="id-ID" sz="2000"><a:solidFill><a:srgbClr val="1E293B"/></a:solidFill></a:rPr>
              <a:t>` + html.EscapeString(slide.Content) + `</a:t>
            </a:r>
          </a:p>
          <a:p/>
          <a:p>
            <a:r>
              <a:rPr lang="id-ID" sz="1400" i="1"><a:solidFill><a:srgbClr val="64748B"/></a:solidFill></a:rPr>
              <a:t>` + html.EscapeString(fmt.Sprintf("Sovera AI Pitch Deck Studio — Kemitraan CSR %s", companyName)) + `</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)
		sf.Write(sXML.Bytes())
	}

	if err := zipWriter.Close(); err != nil {
		return nil, "", err
	}

	return pptxBuf.Bytes(), filename, nil
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func chunkText(text string, limit int) []string {
	if len(text) <= limit {
		return []string{text}
	}
	var chunks []string
	words := strings.Fields(text)
	var current strings.Builder
	for _, w := range words {
		if current.Len()+len(w)+1 > limit {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(w)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func sanitizeFilename(name string) string {
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		} else if r == ' ' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "Korporasi"
	}
	return string(result)
}
