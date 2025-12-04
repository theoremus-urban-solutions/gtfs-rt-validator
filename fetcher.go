package validator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// IsURL returns true if the input looks like an HTTP/HTTPS URL
func IsURL(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}

// FetchInput reads data from either a URL or local file path.
// It automatically detects the input type based on the prefix.
func FetchInput(input string) ([]byte, error) {
	if IsURL(input) {
		return fetchURL(input)
	}
	return os.ReadFile(input)
}

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
