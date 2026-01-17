package bot

import (
	"context"
	"fmt"
	"net/http"
)

// fetchPictureMIMEType fetches the MIME type of a picture URL via GET request.
// Uses /small suffix to minimize data transfer. Falls back to image/jpeg if
// Content-Type is not available.
func (h *Handler) fetchPictureMIMEType(ctx context.Context, url string) (string, error) {
	// Use /small suffix to minimize data transfer
	smallURL := url + "/small"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, smallURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
		h.logger.WarnContext(ctx, "Content-Type header missing, falling back to image/jpeg")
	}

	return mimeType, nil
}
