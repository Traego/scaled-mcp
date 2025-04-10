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

	validCharsRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := GenerateSecureID(tc.length)

			require.NoError(t, err)

			assert.Equal(t, tc.length, len(id))

			assert.True(t, validCharsRegex.MatchString(id), "ID contains invalid characters: %s", id)

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

	validCharsRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := MustGenerateSecureID(tc.length)

			assert.Equal(t, tc.length, len(id))

			assert.True(t, validCharsRegex.MatchString(id), "ID contains invalid characters: %s", id)

			if tc.length > 0 {
				anotherId := MustGenerateSecureID(tc.length)
				assert.NotEqual(t, id, anotherId, "Generated IDs should be different")
			}
		})
	}
}

func TestMustGenerateSecureID_Signature(t *testing.T) {
	result := MustGenerateSecureID(10)
	assert.IsType(t, "", result)
	require.Len(t, result, 10)
}
