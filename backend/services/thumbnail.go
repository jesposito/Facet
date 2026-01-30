package services

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
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

	// Open and decode the source image
	srcImage, err := imaging.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Resize to fit within size x size while preserving aspect ratio
	thumb := imaging.Fit(srcImage, size, size, imaging.Lanczos)

	// Save the thumbnail as JPEG
	if err := imaging.Save(thumb, dstPath); err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return thumbFilename, nil
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
		"image/tiff": true,
	}
	return supportedMimes[strings.ToLower(mimeType)]
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
func GetImageDimensions(filePath string) (width, height int, err error) {
	f, err := os.Open(filePath)
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
