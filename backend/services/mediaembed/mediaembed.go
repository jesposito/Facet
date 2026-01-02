package mediaembed

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// Normalized represents a parsed external media link.
type Normalized struct {
	Provider     string
	EmbedURL     string
	ThumbnailURL string
	Mime         string
}

// Normalize inspects a URL and optional mime/thumbnail to produce embed info.
// We avoid network calls; only static parsing of known providers.
func Normalize(raw, mime, thumb string) Normalized {
	out := Normalized{
		Provider:     "link",
		Mime:         mime,
		ThumbnailURL: thumb,
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return out
	}

	host := strings.ToLower(u.Host)
	pathParts := strings.Split(strings.Trim(strings.TrimPrefix(u.Path, "/"), "/"), "/")

	switch {
	case strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be"):
		id := extractYouTubeID(u)
		if id != "" {
			out.Provider = "youtube"
			out.EmbedURL = "https://www.youtube.com/embed/" + id + "?rel=0"
			out.ThumbnailURL = "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
			out.Mime = "text/html"
			return out
		}
	case strings.Contains(host, "vimeo.com"):
		if len(pathParts) > 0 {
			id := pathParts[len(pathParts)-1]
			out.Provider = "vimeo"
			out.EmbedURL = "https://player.vimeo.com/video/" + id
			out.Mime = "text/html"
			return out
		}
	case strings.Contains(host, "loom.com"):
		if len(pathParts) > 1 && (pathParts[0] == "share" || pathParts[0] == "embed") {
			id := pathParts[1]
			out.Provider = "loom"
			out.EmbedURL = "https://www.loom.com/embed/" + id
			out.Mime = "text/html"
			return out
		}
	case strings.Contains(host, "soundcloud.com"):
		out.Provider = "soundcloud"
		out.EmbedURL = "https://w.soundcloud.com/player/?url=" + url.QueryEscape(raw)
		out.Mime = "text/html"
		return out
	case strings.Contains(host, "open.spotify.com"):
		if len(pathParts) >= 2 {
			typ, id := pathParts[0], pathParts[1]
			out.Provider = "spotify"
			out.EmbedURL = "https://open.spotify.com/embed/" + typ + "/" + id
			out.Mime = "text/html"
			return out
		}
	case strings.Contains(host, "codepen.io"):
		if len(pathParts) >= 3 && pathParts[1] == "pen" {
			user := pathParts[0]
			id := pathParts[2]
			out.Provider = "codepen"
			out.EmbedURL = "https://codepen.io/" + user + "/embed/" + id + "?default-tab=html,result"
			out.Mime = "text/html"
			return out
		}
	case strings.Contains(host, "figma.com"):
		out.Provider = "figma"
		out.EmbedURL = "https://www.figma.com/embed?embed_host=share&url=" + url.QueryEscape(raw)
		out.Mime = "text/html"
		return out
	case strings.Contains(host, "immich"):
		// Self-hosted Immich shares; safest is a link card. Try to inline direct images/videos.
		out.Provider = "immich"
		ext := strings.ToLower(filepath.Ext(u.Path))
		if isImageExt(ext) {
			out.Mime = mimeOr("image/"+strings.TrimPrefix(ext, "."), mime)
			out.EmbedURL = raw
			out.ThumbnailURL = raw
		} else if isVideoExt(ext) {
			out.Mime = mimeOr("video/"+strings.TrimPrefix(ext, "."), mime)
			out.EmbedURL = raw
		}
		return out
	}

	ext := strings.ToLower(path.Ext(u.Path))
	if isImageExt(ext) {
		out.Provider = "image"
		out.Mime = mimeOr("image/"+strings.TrimPrefix(ext, "."), mime)
		out.EmbedURL = raw
		out.ThumbnailURL = raw
		return out
	}
	if isVideoExt(ext) {
		out.Provider = "video"
		out.Mime = mimeOr("video/"+strings.TrimPrefix(ext, "."), mime)
		out.EmbedURL = raw
		return out
	}
	if ext == ".pdf" {
		out.Provider = "pdf"
		out.Mime = mimeOr("application/pdf", mime)
		out.EmbedURL = raw
		return out
	}

	// Default link card
	return out
}

func extractYouTubeID(u *url.URL) string {
	if strings.Contains(u.Host, "youtu.be") {
		return strings.TrimPrefix(u.Path, "/")
	}
	if strings.Contains(u.Host, "youtube.com") {
		q := u.Query().Get("v")
		if q != "" {
			return q
		}
		if strings.HasPrefix(u.Path, "/embed/") {
			return strings.TrimPrefix(u.Path, "/embed/")
		}
	}
	return ""
}

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return true
	default:
		return false
	}
}

func isVideoExt(ext string) bool {
	switch ext {
	case ".mp4", ".webm", ".mov":
		return true
	default:
		return false
	}
}

func mimeOr(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
