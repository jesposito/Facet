package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StorageConfig holds configuration for the storage service
type StorageConfig struct {
	// UploadsDir is the primary directory for user uploads (e.g., /uploads in Docker)
	// If empty or doesn't exist, falls back to DataDir/storage
	UploadsDir string

	// DataDir is the PocketBase data directory (e.g., /data/pb_data)
	// Used as fallback storage location: DataDir/storage
	DataDir string
}

// StorageService manages file storage paths with multi-location support.
// It enables graceful migration from the old storage location (pb_data/storage)
// to a dedicated uploads directory while maintaining access to existing files.
type StorageService struct {
	primaryPath  string // Primary storage: /uploads (if configured)
	fallbackPath string // Fallback storage: {dataDir}/storage (always available)
	hasPrimary   bool   // Whether primary storage is configured and accessible

	// Cache for path existence checks to avoid repeated stat calls
	mu         sync.RWMutex
	existsCache map[string]string // relativePath -> actualFullPath
}

// NewStorageService creates a storage service with optional primary and required fallback paths.
// The primary path (UploadsDir) is used for new uploads if available.
// The fallback path (DataDir/storage) is always available and used for legacy files.
func NewStorageService(cfg StorageConfig) *StorageService {
	s := &StorageService{
		fallbackPath: filepath.Join(cfg.DataDir, "storage"),
		existsCache:  make(map[string]string),
	}

	// Ensure fallback directory exists
	if err := os.MkdirAll(s.fallbackPath, 0755); err != nil {
		// Log but don't fail - PocketBase may create it later
		fmt.Printf("[Storage] Warning: could not create fallback storage directory: %v\n", err)
	}

	// Configure primary path if UploadsDir is set and accessible
	if cfg.UploadsDir != "" {
		// Clean and normalize the path
		uploadsDir := filepath.Clean(cfg.UploadsDir)

		// Check if the directory exists and is accessible
		if info, err := os.Stat(uploadsDir); err == nil && info.IsDir() {
			s.primaryPath = uploadsDir
			s.hasPrimary = true
			fmt.Printf("[Storage] Primary storage configured: %s\n", uploadsDir)
			fmt.Printf("[Storage] Fallback storage: %s\n", s.fallbackPath)
		} else if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(uploadsDir, 0755); err == nil {
				s.primaryPath = uploadsDir
				s.hasPrimary = true
				fmt.Printf("[Storage] Primary storage created: %s\n", uploadsDir)
				fmt.Printf("[Storage] Fallback storage: %s\n", s.fallbackPath)
			} else {
				fmt.Printf("[Storage] Warning: could not create primary storage %s: %v\n", uploadsDir, err)
				fmt.Printf("[Storage] Using fallback storage only: %s\n", s.fallbackPath)
			}
		} else {
			fmt.Printf("[Storage] Warning: primary storage %s not accessible: %v\n", uploadsDir, err)
			fmt.Printf("[Storage] Using fallback storage only: %s\n", s.fallbackPath)
		}
	} else {
		fmt.Printf("[Storage] No UPLOADS_DIR configured, using default: %s\n", s.fallbackPath)
	}

	return s
}

// GetWritePath returns the path where new files should be written.
// Uses primary storage if available, otherwise fallback.
func (s *StorageService) GetWritePath(collectionID, recordID, filename string) string {
	base := s.fallbackPath
	if s.hasPrimary {
		base = s.primaryPath
	}
	return filepath.Join(base, collectionID, recordID, filename)
}

// GetWriteDir returns the directory where new files should be written (without filename).
// Ensures the directory exists.
func (s *StorageService) GetWriteDir(collectionID, recordID string) (string, error) {
	base := s.fallbackPath
	if s.hasPrimary {
		base = s.primaryPath
	}
	dir := filepath.Join(base, collectionID, recordID)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	return dir, nil
}

// FindFile locates an existing file, checking primary storage first, then fallback.
// Returns the full path and true if found, empty string and false if not found.
func (s *StorageService) FindFile(collectionID, recordID, filename string) (string, bool) {
	relPath := filepath.Join(collectionID, recordID, filename)

	// Check cache first
	s.mu.RLock()
	if cached, ok := s.existsCache[relPath]; ok {
		s.mu.RUnlock()
		// Verify cached path still exists
		if _, err := os.Stat(cached); err == nil {
			return cached, true
		}
		// Cache entry invalid, remove it
		s.mu.Lock()
		delete(s.existsCache, relPath)
		s.mu.Unlock()
	} else {
		s.mu.RUnlock()
	}

	// Check primary storage first
	if s.hasPrimary {
		path := filepath.Join(s.primaryPath, relPath)
		if _, err := os.Stat(path); err == nil {
			s.cacheResult(relPath, path)
			return path, true
		}
	}

	// Check fallback storage
	path := filepath.Join(s.fallbackPath, relPath)
	if _, err := os.Stat(path); err == nil {
		s.cacheResult(relPath, path)
		return path, true
	}

	return "", false
}

// FindFileDir locates the directory containing a record's files.
// Returns the full path and true if found, empty string and false if not found.
func (s *StorageService) FindFileDir(collectionID, recordID string) (string, bool) {
	relPath := filepath.Join(collectionID, recordID)

	// Check primary storage first
	if s.hasPrimary {
		path := filepath.Join(s.primaryPath, relPath)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, true
		}
	}

	// Check fallback storage
	path := filepath.Join(s.fallbackPath, relPath)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path, true
	}

	return "", false
}

// DeleteFile removes a file from storage, checking both locations.
// Returns nil if the file was deleted or didn't exist.
func (s *StorageService) DeleteFile(collectionID, recordID, filename string) error {
	path, found := s.FindFile(collectionID, recordID, filename)
	if !found {
		return nil // File doesn't exist, nothing to delete
	}

	// Clear from cache
	relPath := filepath.Join(collectionID, recordID, filename)
	s.mu.Lock()
	delete(s.existsCache, relPath)
	s.mu.Unlock()

	return os.Remove(path)
}

// GetStorageRoots returns all storage directories that should be scanned for files.
// Primary is returned first (if configured), then fallback.
func (s *StorageService) GetStorageRoots() []string {
	if s.hasPrimary && s.primaryPath != s.fallbackPath {
		return []string{s.primaryPath, s.fallbackPath}
	}
	return []string{s.fallbackPath}
}

// GetPrimaryRoot returns the primary storage root, or empty string if not configured.
func (s *StorageService) GetPrimaryRoot() string {
	if s.hasPrimary {
		return s.primaryPath
	}
	return ""
}

// GetFallbackRoot returns the fallback storage root.
func (s *StorageService) GetFallbackRoot() string {
	return s.fallbackPath
}

// IsPrimaryConfigured returns whether a dedicated uploads directory is active.
func (s *StorageService) IsPrimaryConfigured() bool {
	return s.hasPrimary
}

// ResolveRelativePath converts a relative path (collectionID/recordID/filename) to an absolute path.
// Returns the path where the file exists, or where it would be written if it doesn't exist.
func (s *StorageService) ResolveRelativePath(relPath string) string {
	// Check if file exists somewhere
	if s.hasPrimary {
		path := filepath.Join(s.primaryPath, relPath)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	path := filepath.Join(s.fallbackPath, relPath)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// File doesn't exist, return write path (primary if available)
	base := s.fallbackPath
	if s.hasPrimary {
		base = s.primaryPath
	}
	return filepath.Join(base, relPath)
}

// GetStorageInfo returns diagnostic information about storage configuration.
func (s *StorageService) GetStorageInfo() map[string]any {
	info := map[string]any{
		"fallback_path":      s.fallbackPath,
		"primary_configured": s.hasPrimary,
	}

	if s.hasPrimary {
		info["primary_path"] = s.primaryPath
	}

	// Check if paths exist and are writable
	if _, err := os.Stat(s.fallbackPath); err == nil {
		info["fallback_exists"] = true
	} else {
		info["fallback_exists"] = false
	}

	if s.hasPrimary {
		if _, err := os.Stat(s.primaryPath); err == nil {
			info["primary_exists"] = true
		} else {
			info["primary_exists"] = false
		}
	}

	return info
}

// ClearCache clears the file existence cache.
// Useful after bulk operations or migrations.
func (s *StorageService) ClearCache() {
	s.mu.Lock()
	s.existsCache = make(map[string]string)
	s.mu.Unlock()
}

// cacheResult stores a path lookup result in the cache.
func (s *StorageService) cacheResult(relPath, fullPath string) {
	s.mu.Lock()
	// Limit cache size to prevent unbounded growth
	if len(s.existsCache) > 10000 {
		// Clear half the cache when it gets too big
		newCache := make(map[string]string, 5000)
		count := 0
		for k, v := range s.existsCache {
			if count >= 5000 {
				break
			}
			newCache[k] = v
			count++
		}
		s.existsCache = newCache
	}
	s.existsCache[relPath] = fullPath
	s.mu.Unlock()
}
