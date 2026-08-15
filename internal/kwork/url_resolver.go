package kwork

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"kwork-assistant/internal/domain"
)

var (
	resolverClient = &http.Client{
		Timeout: 15 * time.Second,
	}
	punctRegex = regexp.MustCompile(`[^\w\sа-яА-ЯёЁ]+`)
)

func normalizeTitle(t string) string {
	t = html.UnescapeString(t)
	t = strings.ToLower(t)
	t = punctRegex.ReplaceAllString(t, " ")
	return strings.Join(strings.Fields(t), " ")
}

// ResolveProjectURL tries to validate possible project URLs.
// Returns canonical URL if valid, empty string otherwise.
func ResolveProjectURL(ctx context.Context, proj domain.Project) string {
	if proj.ExternalID == 0 {
		return ""
	}
	
	candidates := []string{
		fmt.Sprintf("https://kwork.ru/projects/%d/view", proj.ExternalID),
		fmt.Sprintf("https://kwork.ru/projects/%d", proj.ExternalID),
	}
	
	normTitle := normalizeTitle(proj.Title)
	
	for _, cand := range candidates {
		req, err := http.NewRequestWithContext(ctx, "GET", cand, nil)
		if err != nil {
			continue
		}
		
		// Normal User-Agent
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		
		resp, err := resolverClient.Do(req)
		if err != nil {
			continue
		}
		
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		
		// final host must be kwork.ru
		if resp.Request.URL.Host != "kwork.ru" {
			resp.Body.Close()
			continue
		}
		
		// Check path to avoid redirect to generic homepage/listing
		path := resp.Request.URL.Path
		if path == "/" || path == "/projects" || path == "/projects/" {
			resp.Body.Close()
			continue
		}
		
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024*5)) // max 5MB
		resp.Body.Close()
		if err != nil {
			continue
		}
		
		bodyStr := string(bodyBytes)
		normBody := normalizeTitle(bodyStr)
		
		// 1. Strong Title Match
		if len(normTitle) > 5 && strings.Contains(normBody, normTitle) {
			return resp.Request.URL.String()
		}
		
		// 2. Structured ID marker match
		structuredMarkers := []string{
			fmt.Sprintf(`project_id":%d`, proj.ExternalID),
			fmt.Sprintf(`project_id":"%d"`, proj.ExternalID),
			fmt.Sprintf(`data-project-id="%d"`, proj.ExternalID),
			fmt.Sprintf(`data-id="%d"`, proj.ExternalID),
			fmt.Sprintf(`"id":%d`, proj.ExternalID),
		}
		
		for _, marker := range structuredMarkers {
			if strings.Contains(bodyStr, marker) {
				return resp.Request.URL.String()
			}
		}
	}
	
	return ""
}

// SetResolverClient allows injecting a custom HTTP client for testing
func SetResolverClient(c *http.Client) {
	resolverClient = c
}
