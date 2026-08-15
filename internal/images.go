package internal

import "strings"

type ImageService struct {
	BaseURL string
}

func NewImageService(baseURL string) *ImageService {
	return &ImageService{
		BaseURL: baseURL,
	}
}

// GetFullURL converts a relative path to a full URL.
func (is *ImageService) GetFullURL(relativePath string) string {
	rel := strings.TrimSpace(relativePath)
	if rel == "" {
		return ""
	}
	rel = strings.TrimLeft(rel, "/")
	base := strings.TrimRight(strings.TrimSpace(is.BaseURL), "/")
	if base == "" {
		return rel
	}
	return base + "/" + rel
}
