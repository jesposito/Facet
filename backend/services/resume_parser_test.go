package services

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"strings"
	"testing"
	"time"
)

const docxMIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

type docxPart struct {
	name string
	body string
}

// buildDOCX writes the parts into a ZIP archive the way Word does.
func buildDOCX(t *testing.T, parts ...docxPart) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, part := range parts {
		w, err := zw.Create(part.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(part.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// minimalDOCX returns the parts of a small but complete DOCX whose body is
// documentXML.
func minimalDOCX(documentXML string) []docxPart {
	return []docxPart{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`</Relationships>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`},
		{"word/document.xml", documentXML},
	}
}

const wordNS = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`

// withDeadline fails the test instead of hanging if extraction never returns.
func withDeadline(t *testing.T, extract func() (string, error)) (string, error) {
	t.Helper()
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := extract()
		done <- result{text, err}
	}()
	select {
	case r := <-done:
		return r.text, r.err
	case <-time.After(20 * time.Second):
		t.Fatal("DOCX extraction did not finish within 20s")
		return "", nil
	}
}

func extractDOCXWithDeadline(t *testing.T, data []byte) (string, error) {
	t.Helper()
	return withDeadline(t, func() (string, error) {
		return NewResumeParser(nil).ExtractText(data, docxMIME)
	})
}

func TestResumeDOCXTextExtraction(t *testing.T) {
	data := buildDOCX(t, minimalDOCX(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
		`<w:document `+wordNS+`><w:body>`+
		`<w:p><w:r><w:t>Facet DOCX import regression</w:t></w:r></w:p>`+
		`<w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p>`+
		`</w:body></w:document>`)...)

	text, err := extractDOCXWithDeadline(t, data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Facet DOCX import regression") || !strings.Contains(text, "Second paragraph") {
		t.Fatalf("DOCX text was not preserved: %q", text)
	}
}

func TestResumeDOCXXMLFallbackPreservesText(t *testing.T) {
	text, err := withDeadline(t, func() (string, error) {
		return NewResumeParser(nil).extractDOCXFromXML([]byte(`<w:document ` + wordNS + `><w:body>` +
			`<w:p><w:r><w:t>First</w:t></w:r></w:p><w:p><w:r><w:t>Second</w:t></w:r></w:p>` +
			`</w:body></w:document>`))
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != "\nFirst\nSecond" {
		t.Fatalf("fallback text = %q", text)
	}
}

func TestResumeDOCXMalformedXMLReturnsError(t *testing.T) {
	malformed := []string{
		// Truncated mid-document: the decoder reports unexpected EOF as a
		// syntax error, which used to make the fallback loop forever.
		`<w:document ` + wordNS + `><w:body><w:p><w:r><w:t>Hello</w:t></w:r></w:p><w:p><`,
		// Mismatched closing tag.
		`<w:document ` + wordNS + `><w:body><w:p><w:r><w:t>Hello</w:r></w:p></w:body></w:document>`,
		// Broken markup inside a text run.
		`<w:document ` + wordNS + `><w:body><w:p><w:r><w:t>Hello <<< world</w:t></w:r></w:p></w:body></w:document>`,
	}
	for i, documentXML := range malformed {
		t.Run(fmt.Sprintf("fallback/%d", i), func(t *testing.T) {
			_, err := withDeadline(t, func() (string, error) {
				return NewResumeParser(nil).extractDOCXFromXML([]byte(documentXML))
			})
			if err == nil || !strings.Contains(err.Error(), "malformed") {
				t.Fatalf("err = %v, want malformed XML error", err)
			}
		})
		t.Run(fmt.Sprintf("upload/%d", i), func(t *testing.T) {
			_, err := extractDOCXWithDeadline(t, buildDOCX(t, minimalDOCX(documentXML)...))
			if err == nil {
				t.Fatal("malformed DOCX was accepted")
			}
		})
	}
}

func TestResumeDOCXRejectsOversizedUpload(t *testing.T) {
	data := make([]byte, docxMaxCompressedBytes+1)
	_, err := extractDOCXWithDeadline(t, data)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("err = %v, want too large error", err)
	}
}

func TestResumeDOCXRejectsTooManyEntries(t *testing.T) {
	parts := minimalDOCX(`<w:document ` + wordNS + `><w:body><w:p><w:r><w:t>Hi</w:t></w:r></w:p></w:body></w:document>`)
	for i := 0; len(parts) <= docxMaxEntries; i++ {
		parts = append(parts, docxPart{fmt.Sprintf("word/media/extra%d.xml", i), ""})
	}
	_, err := extractDOCXWithDeadline(t, buildDOCX(t, parts...))
	if err == nil || !strings.Contains(err.Error(), "too many parts") {
		t.Fatalf("err = %v, want too many parts error", err)
	}
}

func TestResumeDOCXRejectsLargeDocumentXML(t *testing.T) {
	// Highly repetitive XML compresses to a tiny upload but expands past the
	// document.xml limit.
	run := `<w:p><w:r><w:t>aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa</w:t></w:r></w:p>`
	body := strings.Repeat(run, docxMaxDocumentXMLBytes/len(run)+1)
	documentXML := `<w:document ` + wordNS + `><w:body>` + body + `</w:body></w:document>`

	data := buildDOCX(t, minimalDOCX(documentXML)...)
	if len(data) > docxMaxCompressedBytes {
		t.Fatalf("test archive is %d bytes; it should be small enough to pass the upload limit", len(data))
	}
	_, err := extractDOCXWithDeadline(t, data)
	if err == nil || !strings.Contains(err.Error(), "too large when uncompressed") {
		t.Fatalf("err = %v, want uncompressed size error", err)
	}
}

// rawDeflatePart adds a part whose ZIP header declares declaredSize while the
// compressed data actually expands to actualSize bytes of zeros.
func rawDeflatePart(t *testing.T, zw *zip.Writer, name string, declaredSize uint64, actualSize int) {
	t.Helper()
	var compressed bytes.Buffer
	fw, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(make([]byte, actualSize)); err != nil {
		t.Fatal(err)
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	w, err := zw.CreateRaw(&zip.FileHeader{
		Name:               name,
		Method:             zip.Deflate,
		CompressedSize64:   uint64(compressed.Len()),
		UncompressedSize64: declaredSize,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(compressed.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestResumeDOCXRejectsZipBombs(t *testing.T) {
	documentXML := `<w:document ` + wordNS + `><w:body><w:p><w:r><w:t>Hi</w:t></w:r></w:p></w:body></w:document>`

	cases := map[string]struct {
		declared uint64
		actual   int
	}{
		// Honest headers: rejected from the declared sizes alone.
		"declared": {declared: 1 << 30, actual: 1 << 20},
		// Headers that understate the real size: rejected while reading,
		// because the real bytes are counted.
		"understated": {declared: 1024, actual: 8 << 20},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			for _, part := range minimalDOCX(documentXML) {
				w, err := zw.Create(part.name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte(part.body)); err != nil {
					t.Fatal(err)
				}
			}
			rawDeflatePart(t, zw, "word/media/bomb.bin", tc.declared, tc.actual)
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}

			_, err := extractDOCXWithDeadline(t, buf.Bytes())
			if err == nil {
				t.Fatal("zip bomb was accepted")
			}
		})
	}
}

// Exercise the real PDF decoder used for resume imports after go-fitz upgrades.
func TestResumePDFTextExtraction(t *testing.T) {
	content := "BT /F1 12 Tf 72 720 Td (Facet resume import regression) Tj ET\n"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)

	text, err := NewResumeParser(nil).ExtractText(pdf.Bytes(), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Facet resume import regression") {
		t.Fatalf("PDF text was not preserved: %q", text)
	}
}
