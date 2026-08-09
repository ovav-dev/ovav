// Package scraper implements OVAV's web scraping for research targets.
package scraper

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ovav/ovav/internal/autonomous/scheduler"
)

// Client is an HTTP client for scraping research targets.
type Client struct {
	client  *http.Client
	timeout time.Duration
	headers map[string]string
}

// New creates a new scraper client.
func New(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		headers: map[string]string{
			"User-Agent":      "OVAV-ResearchBot/1.0",
			"Accept":          "text/html,application/xhtml+xml",
			"Accept-Language": "en-US,en;q=0.9",
		},
	}
}

// Scrape fetches content from a research target URL.
func (c *Client) Scrape(target *scheduler.Target) (string, error) {
	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", target.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, target.URL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}

// ScrapeMultiple fetches content from multiple URLs.
func (c *Client) ScrapeMultiple(targets []scheduler.Target) (map[string]string, []error) {
	results := make(map[string]string)
	var errs []error

	for _, t := range targets {
		content, err := c.Scrape(&t)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		results[t.ID] = content
	}

	return results, errs
}
