package kwork

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"kwork-assistant/internal/domain"
)

type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestResolveProjectURL(t *testing.T) {
	var mockStatus int
	var mockBody string
	var mockFinalHost string
	var mockFinalPath string

	transport := &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: mockStatus,
				Body:       io.NopCloser(bytes.NewBufferString(mockBody)),
				Request:    req,
			}
			
			// Overwrite the final URL if mocked
			if mockFinalHost != "" {
				resp.Request.URL.Host = mockFinalHost
			}
			if mockFinalPath != "" {
				resp.Request.URL.Path = mockFinalPath
			}
			
			return resp, nil
		},
	}

	SetResolverClient(&http.Client{Transport: transport})

	ctx := context.Background()
	proj := domain.Project{
		ExternalID: 12345,
		Title:      "Test Project Title",
	}

	tests := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "A. 200 + exact/normalized title => VALID",
			setup: func() {
				mockStatus = 200
				mockBody = `<html><body>Some text. test project title inside body.</body></html>`
				mockFinalHost = "kwork.ru"
				mockFinalPath = "/projects/12345/view"
			},
			expected: true,
		},
		{
			name: "B. 200 + external ID somewhere, no title => INVALID",
			setup: func() {
				mockStatus = 200
				mockBody = `<html><body>Some generic text. Here is a random number 12345.</body></html>`
				mockFinalHost = "kwork.ru"
				mockFinalPath = "/projects/12345/view"
			},
			expected: false,
		},
		{
			name: "B.2 200 + structured project_id => VALID",
			setup: func() {
				mockStatus = 200
				mockBody = `<html><body><div data-project-id="12345"></div></body></html>`
				mockFinalHost = "kwork.ru"
				mockFinalPath = "/projects/12345/view"
			},
			expected: true,
		},
		{
			name: "C. redirect to /projects => INVALID",
			setup: func() {
				mockStatus = 200
				mockFinalHost = "kwork.ru"
				mockFinalPath = "/projects"
			},
			expected: false,
		},
		{
			name: "D. redirect to / => INVALID",
			setup: func() {
				mockStatus = 200
				mockFinalHost = "kwork.ru"
				mockFinalPath = "/"
			},
			expected: false,
		},
		{
			name: "E. wrong host => INVALID",
			setup: func() {
				mockStatus = 200
				mockFinalHost = "other.ru"
				mockFinalPath = "/projects/12345/view"
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			url := ResolveProjectURL(ctx, proj)
			if tc.expected && url == "" {
				t.Errorf("Expected canonical URL, got empty")
			}
			if !tc.expected && url != "" {
				t.Errorf("Expected empty URL, got %s", url)
			}
		})
	}
}
