package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecureID(t *testing.T) {
	testCases := []struct {
		name   string
		length int
	}{
		{
			name:   "zero length",
			length: 0,
		},
		{
			name:   "small length",
			length: 8,
		},
		{
			name:   "medium length",
			length: 16,
		},
		{
			name:   "large length",
			length: 32,
		},
	}

	// Regex to validate the ID contains only allowed characters
	validCharsRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate an ID
			id, err := GenerateSecureID(tc.length)

			// Check there's no error
			require.NoError(t, err)

			// Check the length is correct
			assert.Equal(t, tc.length, len(id))

			// Check the ID contains only valid characters
			assert.True(t, validCharsRegex.MatchString(id), "ID contains invalid characters: %s", id)

			// Generate another ID to ensure they're different (very low probability of collision)
			if tc.length > 0 {
				anotherId, err := GenerateSecureID(tc.length)
				require.NoError(t, err)
				assert.NotEqual(t, id, anotherId, "Generated IDs should be different")
			}
		})
	}
}

func TestMustGenerateSecureID(t *testing.T) {
	testCases := []struct {
		name   string
		length int
	}{
		{
			name:   "zero length",
			length: 0,
		},
		{
			name:   "small length",
			length: 8,
		},
		{
			name:   "medium length",
			length: 16,
		},
		{
			name:   "large length",
			length: 32,
		},
	}

	// Regex to validate the ID contains only allowed characters
	validCharsRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This should not panic
			id := MustGenerateSecureID(tc.length)

			// Check the length is correct
			assert.Equal(t, tc.length, len(id))

			// Check the ID contains only valid characters
			assert.True(t, validCharsRegex.MatchString(id), "ID contains invalid characters: %s", id)

			// Generate another ID to ensure they're different (very low probability of collision)
			if tc.length > 0 {
				anotherId := MustGenerateSecureID(tc.length)
				assert.NotEqual(t, id, anotherId, "Generated IDs should be different")
			}
		})
	}
}

// Test that MustGenerateSecureID panics when there's an error
// Note: This is difficult to test directly since we can't easily cause rand.Int to fail,
// but we can verify the function signature and behavior in normal cases.
func TestMustGenerateSecureID_Signature(t *testing.T) {
	// Verify that the function takes an int and returns a string
	result := MustGenerateSecureID(10)
	assert.IsType(t, "", result)
	require.Len(t, result, 10)
}
