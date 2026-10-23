package services

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

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
