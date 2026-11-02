package services

// Advisory investigation: GHSA-q7pp-wcgr-pffx / CVE-2023-36308
// (github.com/disintegration/imaging#165).
//
// A crafted palette-color TIFF can carry a pixel whose palette index is larger
// than the palette. imaging's scanner does an unchecked lookup
// (scanner.go:242, `c := s.palette[img.Pix[i]]`) and panics with
// "index out of range". imaging reaches TIFF through golang.org/x/image/tiff,
// which is registered globally by importing imaging, so the file thumbnail
// route can decode a TIFF even though Facet's own ThumbnailService excludes
// image/tiff from its MIME allowlist.
//
// This test proves, against the exact module versions pinned in go.mod, whether
// modern x/image already neutralises the malformed palette before imaging ever
// sees an out-of-range index. golang/image commit 3bbf4a6 (June 2024,
// "tiff: Validate palette indices when parsing palette-color images") added
// that guard; x/image v0.47.0 post-dates it.
//
// The decode+resize happens in a re-exec'd child process with a hard timeout.
// A panic raised inside imaging's worker goroutines cannot be recovered by the
// parent's deferred recover (nor by PocketBase's routine.SafeWrap, whose
// recover runs only on the calling goroutine), so running it out-of-process is
// the only way to observe the crash without taking the test binary (and the
// supervisor) down with it.

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/disintegration/imaging"
)

const reproChildEnv = "FACET_TIFF_ADVISORY_CHILD"

// buildPaletteTIFF writes a minimal little-endian palette-color TIFF.
// paletteLen is the number of RGB entries actually supplied in the ColorMap.
// pixels are the raw 8-bit palette indices (one byte each, row-major) for a
// width x height image. Passing an index >= paletteLen is what reproduces the
// advisory, because x/image derives the palette length from the ColorMap size.
func buildPaletteTIFF(width, height, paletteLen int, pixels []byte) []byte {
	const headerLen = 8
	const entryCount = 10
	ifdLen := 2 + entryCount*12 + 4 // count + entries + next-IFD offset

	pixelOffset := headerLen + ifdLen
	colorMapOffset := pixelOffset + len(pixels)
	colorMapShorts := paletteLen * 3 // R plane, G plane, B plane

	buf := &bytes.Buffer{}
	le := binary.LittleEndian

	// Header: byte order, magic 42, offset to IFD0.
	buf.WriteString("II")
	writeU16(buf, le, 42)
	writeU32(buf, le, uint32(headerLen))

	// IFD entry count.
	writeU16(buf, le, entryCount)

	const (
		tImageWidth    = 256
		tImageLength   = 257
		tBitsPerSample = 258
		tCompression   = 259
		tPhotometric   = 262
		tStripOffsets  = 273
		tSamplesPerPix = 277
		tRowsPerStrip  = 278
		tStripByteCnts = 279
		tColorMap      = 320
		typeShort      = 3
		typeLong       = 4
	)

	writeShortEntry(buf, le, tImageWidth, uint16(width))
	writeShortEntry(buf, le, tImageLength, uint16(height))
	writeShortEntry(buf, le, tBitsPerSample, 8)
	writeShortEntry(buf, le, tCompression, 1) // no compression
	writeShortEntry(buf, le, tPhotometric, 3) // RGB palette
	writeLongEntry(buf, le, tStripOffsets, uint32(pixelOffset))
	writeShortEntry(buf, le, tSamplesPerPix, 1)
	writeShortEntry(buf, le, tRowsPerStrip, uint16(height))
	writeLongEntry(buf, le, tStripByteCnts, uint32(len(pixels)))
	// ColorMap: SHORT, count = 3*paletteLen, stored at colorMapOffset.
	writeEntry(buf, le, tColorMap, typeShort, uint32(colorMapShorts), uint32(colorMapOffset))

	writeU32(buf, le, 0) // next IFD offset: none

	// Strip (pixel indices).
	buf.Write(pixels)

	// ColorMap values: 16-bit, three planes. Values are arbitrary; only the
	// count matters for deriving the palette length.
	for i := 0; i < colorMapShorts; i++ {
		writeU16(buf, le, uint16((i*257)&0xffff))
	}

	return buf.Bytes()
}

func writeU16(buf *bytes.Buffer, le binary.ByteOrder, v uint16) {
	b := make([]byte, 2)
	le.PutUint16(b, v)
	buf.Write(b)
}

func writeU32(buf *bytes.Buffer, le binary.ByteOrder, v uint32) {
	b := make([]byte, 4)
	le.PutUint32(b, v)
	buf.Write(b)
}

func writeEntry(buf *bytes.Buffer, le binary.ByteOrder, tag, typ uint16, count, valueOrOffset uint32) {
	writeU16(buf, le, tag)
	writeU16(buf, le, typ)
	writeU32(buf, le, count)
	writeU32(buf, le, valueOrOffset)
}

// writeShortEntry stores a single SHORT inline (low 2 bytes of the value field).
func writeShortEntry(buf *bytes.Buffer, le binary.ByteOrder, tag uint16, v uint16) {
	writeEntry(buf, le, tag, 3, 1, uint32(v))
}

func writeLongEntry(buf *bytes.Buffer, le binary.ByteOrder, tag uint16, v uint32) {
	writeEntry(buf, le, tag, 4, 1, v)
}

// decodeAndResize mirrors the imaging calls PocketBase's createThumb makes:
// imaging.Decode with AutoOrientation, then a resize that drives the scanner
// (scanner.go), which is where the advisory panics on a bad palette index.
func decodeAndResize(data []byte) (string, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return "decode-error", err
	}
	resized := imaging.Resize(img, 64, 0, imaging.Lanczos)
	if resized == nil {
		return "resize-nil", nil
	}
	return "ok", nil
}

// TestTIFFAdvisoryChild is the out-of-process worker. It is a no-op during a
// normal run and only does the decode when the parent re-execs it with
// reproChildEnv set, so a goroutine panic crashes just this child.
func TestTIFFAdvisoryChild(t *testing.T) {
	mode := os.Getenv(reproChildEnv)
	if mode == "" {
		t.Skip("child-only entry point; driven by TestMalformedPaletteTIFFDoesNotCrash")
	}

	var data []byte
	switch mode {
	case "malformed":
		// 2x2 image, palette of 65 colours, first index 70 -> out of range.
		data = buildPaletteTIFF(2, 2, 65, []byte{70, 0, 0, 0})
	case "valid-palette":
		// Same shape, but every index is inside the 65-colour palette.
		data = buildPaletteTIFF(2, 2, 65, []byte{64, 0, 1, 2})
	default:
		t.Fatalf("unknown child mode %q", mode)
	}

	outcome, err := decodeAndResize(data)
	// Reaching here means no panic. Report what happened on stdout for the
	// parent; the exit code (0) is the signal that the process survived.
	if err != nil {
		os.Stdout.WriteString("SURVIVED outcome=" + outcome + " err=" + err.Error() + "\n")
	} else {
		os.Stdout.WriteString("SURVIVED outcome=" + outcome + "\n")
	}
}

// runChild re-execs this test binary to run only TestTIFFAdvisoryChild in the
// given mode, with a hard timeout. It returns whether the child exited cleanly
// (no panic) plus its combined output.
func runChild(t *testing.T, mode string) (survived bool, output string) {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot find test executable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestTIFFAdvisoryChild$", "-test.v")
	cmd.Env = append(os.Environ(), reproChildEnv+"="+mode)
	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("child timed out in mode %q (possible hang); output:\n%s", mode, out)
	}
	return err == nil, string(out)
}

// TestMalformedPaletteTIFFDoesNotCrash is the real assertion. With the pinned
// x/image v0.47.0 the malformed palette must not panic the decode/resize path.
// If this ever starts failing (child crashes), the advisory is live against
// our dependency set and the recommended mitigation in FACET-PATCH territory
// must change from "documented, already fixed upstream" to an active patch.
func TestMalformedPaletteTIFFDoesNotCrash(t *testing.T) {
	survived, output := runChild(t, "malformed")
	if !survived {
		t.Fatalf("CVE-2023-36308 reproduced: imaging crashed on a malformed palette TIFF "+
			"with pinned x/image.\nChild output:\n%s", output)
	}
	if !strings.Contains(output, "SURVIVED") {
		t.Fatalf("child exited 0 but did not report survival; output:\n%s", output)
	}
	t.Logf("malformed palette TIFF handled without panic:\n%s", strings.TrimSpace(childReport(output)))
}

// TestValidPaletteTIFFStillDecodes confirms the guard does not break ordinary
// palette-color TIFFs: in-range indices decode and resize cleanly.
func TestValidPaletteTIFFStillDecodes(t *testing.T) {
	survived, output := runChild(t, "valid-palette")
	if !survived {
		t.Fatalf("valid palette TIFF unexpectedly crashed:\n%s", output)
	}
	if !strings.Contains(output, "outcome=ok") {
		t.Fatalf("valid palette TIFF did not decode+resize cleanly; output:\n%s", output)
	}
}

// TestNormalImageDecodes is the in-process happy path: a plain RGBA image
// encoded as PNG decodes and resizes with no TIFF involvement at all.
func TestNormalImageDecodes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.Set(x, y, image.White.C)
		}
	}
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, src, imaging.PNG); err != nil {
		t.Fatalf("encode: %v", err)
	}
	outcome, err := decodeAndResize(buf.Bytes())
	if err != nil {
		t.Fatalf("normal image failed to decode+resize: %v", err)
	}
	if outcome != "ok" {
		t.Fatalf("unexpected outcome for normal image: %q", outcome)
	}
}

func childReport(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "SURVIVED") {
			return line
		}
	}
	return output
}
