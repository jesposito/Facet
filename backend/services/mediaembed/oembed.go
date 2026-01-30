package mediaembed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OEmbedMetadata holds fetched metadata from an oEmbed provider.
type OEmbedMetadata struct {
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	Provider     string `json:"provider"`
	Duration     int    `json:"duration,omitempty"` // in seconds, if available
}

// oEmbedResponse is the common subset of fields we parse from oEmbed responses.
type oEmbedResponse struct {
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	Duration     int    `json:"duration,omitempty"`
}

// FetchMetadata attempts to fetch metadata from known oEmbed providers.
// Returns metadata if successful, or an error if fetching fails or provider is unsupported.
// The timeout is 5 seconds to avoid blocking the UI.
func FetchMetadata(rawURL string) (*OEmbedMetadata, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(u.Host)

	// Determine which oEmbed endpoint to use
	var oembedURL string
	var provider string

	switch {
	case strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be"):
		provider = "youtube"
		oembedURL = "https://www.youtube.com/oembed?url=" + url.QueryEscape(rawURL) + "&format=json"

	case strings.Contains(host, "vimeo.com"):
		provider = "vimeo"
		oembedURL = "https://vimeo.com/api/oembed.json?url=" + url.QueryEscape(rawURL)

	case strings.Contains(host, "loom.com"):
		provider = "loom"
		// Extract video ID for Loom's oEmbed
		oembedURL = "https://www.loom.com/v1/oembed?url=" + url.QueryEscape(rawURL)

	case strings.Contains(host, "soundcloud.com"):
		provider = "soundcloud"
		oembedURL = "https://soundcloud.com/oembed?url=" + url.QueryEscape(rawURL) + "&format=json"

	default:
		return nil, fmt.Errorf("unsupported provider for URL: %s", host)
	}

	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", oembedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a user agent to avoid potential blocks
	req.Header.Set("User-Agent", "Facet/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oEmbed request failed with status %d", resp.StatusCode)
	}

	var oembed oEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&oembed); err != nil {
		return nil, fmt.Errorf("failed to parse oEmbed response: %w", err)
	}

	// For YouTube, construct a higher quality thumbnail if we got a video ID
	thumbnailURL := oembed.ThumbnailURL
	if provider == "youtube" && thumbnailURL == "" {
		id := extractYouTubeID(u)
		if id != "" {
			thumbnailURL = "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
		}
	}

	return &OEmbedMetadata{
		Title:        oembed.Title,
		ThumbnailURL: thumbnailURL,
		Provider:     provider,
		Duration:     oembed.Duration,
	}, nil
}

// IsSupportedProvider returns true if the URL belongs to a provider we can fetch metadata from.
func IsSupportedProvider(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Host)
	return strings.Contains(host, "youtube.com") ||
		strings.Contains(host, "youtu.be") ||
		strings.Contains(host, "vimeo.com") ||
		strings.Contains(host, "loom.com") ||
		strings.Contains(host, "soundcloud.com")
}
