package email

import (
	"testing"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple html",
			input:    "<p>Hello <b>World</b></p>",
			expected: "Hello World",
		},
		{
			name:     "with line breaks",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "divs and paragraphs",
			input:    "<div><p>First paragraph</p><p>Second paragraph</p></div>",
			expected: "First paragraph\nSecond paragraph",
		},
		{
			name:     "scripts and styles removed",
			input:    "<style>.red { color: red; }</style><script>alert(1);</script><p>Text</p>",
			expected: "Text",
		},
		{
			name:     "tables",
			input:    "<table><tr><td>Cell 1</td><td>Cell 2</td></tr></table>",
			expected: "Cell 1 Cell 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
