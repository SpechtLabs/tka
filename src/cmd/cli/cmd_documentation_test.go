package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixHeadingLevels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		depth    int
	}{
		{
			name:  "root level",
			depth: 0,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
# Usage tka
## Description
### Theming
### Notes
`,
		},
		{
			name:  "first level",
			depth: 1,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
## Usage tka
### Description
#### Theming
#### Notes
`,
		},
		{
			name:  "third level",
			depth: 3,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
#### Usage tka
##### Description
###### Theming
###### Notes
`,
		},
		{
			name: "hashtag in the middle",
			input: `
# A title with #hashtag in the middle
Bla bla bla`,
			depth: 2,
			expected: `
### A title with #hashtag in the middle
Bla bla bla`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fixHeadingLevels(test.input, test.depth)
			require.Equal(t, test.expected, result)
		})
	}
}
