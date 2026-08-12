package email

import (
	"strings"
	"golang.org/x/net/html"
)

// StripHTML converts HTML to a safe plain text by extracting text nodes
// and inserting appropriate spacing.
func StripHTML(htmlBody string) string {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		// If parse fails, return the string without HTML tags using a basic approach
		return fallbackStripHTML(htmlBody)
	}

	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				b.WriteString(text)
				b.WriteString(" ")
			}
		}
		// Skip scripts and styles
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		// Insert newlines for block elements
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "br" || n.Data == "div") {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		
		// Add newlines after block elements
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "tr" || n.Data == "li") {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n")
			}
		}
	}
	f(doc)

	res := b.String()
	// Clean up spaces before newlines
	res = strings.ReplaceAll(res, " \n", "\n")
	return strings.TrimSpace(res)
}

func fallbackStripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, c := range s {
		if c == '<' {
			inTag = true
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(c)
		}
	}
	return strings.TrimSpace(b.String())
}
