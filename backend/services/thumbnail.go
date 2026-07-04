package services

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ThumbnailSize is the default size for generated thumbnails.
const ThumbnailSize = 200

// ThumbnailSuffix is appended to the original filename for thumbnails.
const ThumbnailSuffix = "_thumb"

// ThumbnailFormat is the output format for thumbnails.
const ThumbnailFormat = ".jpg"

// ThumbnailService handles thumbnail generation for uploaded images.
type ThumbnailService struct {
	storage *StorageService
}

// NewThumbnailService creates a new ThumbnailService.
func NewThumbnailService(storage *StorageService) *ThumbnailService {
	return &ThumbnailService{storage: storage}
}

// GenerateThumbnail creates a thumbnail for the given image file.
// It resizes the image to fit within a square of the specified size while preserving aspect ratio.
// Returns the thumbnail filename on success.
func (ts *ThumbnailService) GenerateThumbnail(collectionID, recordID, filename string, size int) (string, error) {
	if size <= 0 {
		size = ThumbnailSize
	}

	// Find the source file
	srcPath, found := ts.storage.FindFile(collectionID, recordID, filename)
	if !found {
		return "", fmt.Errorf("source file not found: %s/%s/%s", collectionID, recordID, filename)
	}

	// Generate thumbnail filename
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	thumbFilename := nameWithoutExt + ThumbnailSuffix + ThumbnailFormat

	// Determine destination path (same directory as source)
	srcDir := filepath.Dir(srcPath)
	dstPath := filepath.Join(srcDir, thumbFilename)

	// Check if thumbnail already exists
	if _, err := os.Stat(dstPath); err == nil {
		return thumbFilename, nil
	}

	// For HEIC/HEIF files, convert to JPEG first
	imagePath := srcPath
	var tempFile string
	if isHEICFile(filename) {
		converted, err := convertHEICToJPEG(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to convert HEIC: %w", err)
		}
		imagePath = converted
		tempFile = converted
		defer os.Remove(tempFile)
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer f.Close()

	srcImage, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to fit within size x size while preserving aspect ratio
	thumb := resizeToFit(srcImage, size)

	// Save the thumbnail as JPEG
	out, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail: %w", err)
	}
	defer out.Close()
	if err := jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return thumbFilename, nil
}

func resizeToFit(src image.Image, maxSize int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || maxSize <= 0 {
		return src
	}

	scale := math.Min(float64(maxSize)/float64(width), float64(maxSize)/float64(height))
	if scale > 1 {
		scale = 1
	}
	dstWidth := max(1, int(math.Round(float64(width)*scale)))
	dstHeight := max(1, int(math.Round(float64(height)*scale)))
	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// GetThumbnailPath returns the expected thumbnail filename for a given original filename.
func GetThumbnailPath(filename string) string {
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	return nameWithoutExt + ThumbnailSuffix + ThumbnailFormat
}

// ThumbnailExists checks if a thumbnail exists for the given file.
func (ts *ThumbnailService) ThumbnailExists(collectionID, recordID, filename string) bool {
	thumbFilename := GetThumbnailPath(filename)
	_, found := ts.storage.FindFile(collectionID, recordID, thumbFilename)
	return found
}

// GetThumbnailURL returns the API URL for the thumbnail if it exists.
// Returns empty string if thumbnail doesn't exist.
func (ts *ThumbnailService) GetThumbnailURL(collectionID, recordID, filename string) string {
	thumbFilename := GetThumbnailPath(filename)
	if _, found := ts.storage.FindFile(collectionID, recordID, thumbFilename); found {
		// Use custom thumbnail endpoint (PocketBase won't serve unregistered files)
		return fmt.Sprintf("/api/media/thumb/%s/%s/%s", collectionID, recordID, thumbFilename)
	}
	return ""
}

// IsSupportedFormat checks if the file format is supported for thumbnail generation.
func IsSupportedFormat(mimeType string) bool {
	supportedMimes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"image/bmp":  true,
		"image/heic": true,
		"image/heif": true,
	}
	return supportedMimes[strings.ToLower(mimeType)]
}

// IsHEICFormat checks if the mime type is HEIC/HEIF format.
func IsHEICFormat(mimeType string) bool {
	lower := strings.ToLower(mimeType)
	return lower == "image/heic" || lower == "image/heif"
}

// convertHEICToJPEG uses heif-convert (from libheif) to convert HEIC to JPEG.
// Returns the path to the converted JPEG file, which should be cleaned up after use.
func convertHEICToJPEG(heicPath string) (string, error) {
	// Create temp file for output
	tmpDir := os.TempDir()
	outPath := filepath.Join(tmpDir, fmt.Sprintf("heic_convert_%d.jpg", os.Getpid()))

	// Run heif-convert
	cmd := exec.Command("heif-convert", "-q", "90", heicPath, outPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("heif-convert failed: %w", err)
	}

	return outPath, nil
}

// isHEICFile checks if the filename has a HEIC/HEIF extension.
func isHEICFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".heic" || ext == ".heif"
}

// EnsureThumbnail generates a thumbnail if one doesn't exist and the format is supported.
// Returns the thumbnail URL if available, empty string otherwise.
func (ts *ThumbnailService) EnsureThumbnail(collectionID, recordID, filename, mimeType string) string {
	// Check if format is supported
	if !IsSupportedFormat(mimeType) {
		return ""
	}

	// Check if thumbnail already exists
	thumbURL := ts.GetThumbnailURL(collectionID, recordID, filename)
	if thumbURL != "" {
		return thumbURL
	}

	// Generate thumbnail
	thumbFilename, err := ts.GenerateThumbnail(collectionID, recordID, filename, ThumbnailSize)
	if err != nil {
		// Log error but don't fail - thumbnails are optional
		return ""
	}

	// Use custom thumbnail endpoint (PocketBase won't serve unregistered files)
	return fmt.Sprintf("/api/media/thumb/%s/%s/%s", collectionID, recordID, thumbFilename)
}

// DeleteThumbnail removes the thumbnail for a given file.
func (ts *ThumbnailService) DeleteThumbnail(collectionID, recordID, filename string) error {
	thumbFilename := GetThumbnailPath(filename)
	return ts.storage.DeleteFile(collectionID, recordID, thumbFilename)
}

// GetImageDimensions reads the dimensions of an image without loading the full image.
// For HEIC/HEIF files, it converts to JPEG first since Go's standard library doesn't support HEIC.
func GetImageDimensions(filePath string) (width, height int, err error) {
	// For HEIC files, convert to JPEG first to read dimensions
	readPath := filePath
	if isHEICFile(filePath) {
		converted, err := convertHEICToJPEG(filePath)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to convert HEIC for dimensions: %w", err)
		}
		defer os.Remove(converted)
		readPath = converted
	}

	f, err := os.Open(readPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}
