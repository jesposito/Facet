package services

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requirePandoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}
}

func pandocToLaTeX(t *testing.T, from, input string) string {
	t.Helper()
	cmd := exec.Command("pandoc", "-f", from, "-t", "latex", input)
	cmd.Env = pandocEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pandoc: %v\n%s", err, out)
	}
	return string(out)
}

// The sample holds the ways markdown can smuggle TeX to LaTeX: a bare
// command, inline math in both styles and a raw latex block.
func TestResumeMarkdownCannotReachLaTeX(t *testing.T) {
	requirePandoc(t)
	input := filepath.Join("testdata", "resume_injection.md")

	// Pandoc's default reader passes the commands through, so the sample is
	// a real injection.
	if out := pandocToLaTeX(t, "markdown", input); !strings.Contains(out, `\input{/etc/hostname}`) {
		t.Fatalf("default reader no longer passes raw TeX; sample needs updating:\n%s", out)
	}

	out := pandocToLaTeX(t, pandocInputFormat, input)
	if strings.Contains(out, `\input{`) {
		t.Fatalf("raw TeX reached LaTeX:\n%s", out)
	}
	if !strings.Contains(out, `\textbackslash`) {
		t.Fatalf("expected the commands to be printed as text:\n%s", out)
	}
}

// An image path in the AI output must not pull a server file into the DOCX.
func TestResumeDOCXDoesNotEmbedLocalFiles(t *testing.T) {
	requirePandoc(t)

	var img bytes.Buffer
	if err := png.Encode(&img, image.NewGray(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	const marker = "FACET-SECRET-MARKER"
	img.WriteString(marker)
	secret := filepath.Join(t.TempDir(), "secret.png")
	if err := os.WriteFile(secret, img.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	markdown := "# Ada Example\n\n![photo](" + secret + ")\n"

	// Without the sandbox pandoc embeds the file, so the check below means
	// something.
	mdPath := filepath.Join(t.TempDir(), "resume.md")
	outPath := filepath.Join(t.TempDir(), "resume.docx")
	if err := os.WriteFile(mdPath, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("pandoc", mdPath, "-o", outPath).CombinedOutput(); err != nil {
		t.Fatalf("baseline pandoc: %v\n%s", err, out)
	}
	baseline, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !docxContains(t, baseline, marker) {
		t.Fatal("pandoc no longer embeds local images by default; test needs updating")
	}

	docx, err := NewResumeService(nil).convertToDOCX(markdown)
	if err != nil {
		t.Fatalf("convertToDOCX: %v", err)
	}
	if docxContains(t, docx, marker) {
		t.Fatal("the generated DOCX embeds the local file")
	}
}

func docxContains(t *testing.T, docx []byte, marker string) bool {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(rc)
		rc.Close()
		if bytes.Contains(content, []byte(marker)) {
			return true
		}
	}
	return false
}
