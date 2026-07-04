package services

import "testing"

func TestIsSupportedFormatSkipsTIFF(t *testing.T) {
	if IsSupportedFormat("image/tiff") {
		t.Fatal("TIFF thumbnailing should stay disabled while imaging has no patched TIFF advisory fix")
	}
}
