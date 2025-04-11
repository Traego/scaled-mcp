package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSessionActorName(t *testing.T) {
	testCases := []struct {
		name           string
		sessionId      string
		expectedResult string
	}{
		{
			name:           "standard session ID",
			sessionId:      "abc123",
			expectedResult: "abc123-session",
		},
		{
			name:           "empty session ID",
			sessionId:      "",
			expectedResult: "-session",
		},
		{
			name:           "session ID with special characters",
			sessionId:      "user@client_example.com",
			expectedResult: "user@client_example.com-session",
		},
		{
			name:           "numeric session ID",
			sessionId:      "12345",
			expectedResult: "12345-session",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetSessionActorName(tc.sessionId)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestGetDefaultSSEConnectionName(t *testing.T) {
	testCases := []struct {
		name           string
		sessionId      string
		expectedResult string
	}{
		{
			name:           "standard session ID",
			sessionId:      "abc123",
			expectedResult: "abc123-channels-default",
		},
		{
			name:           "empty session ID",
			sessionId:      "",
			expectedResult: "-channels-default",
		},
		{
			name:           "session ID with special characters",
			sessionId:      "user@client_example.com",
			expectedResult: "user@client_example.com-channels-default",
		},
		{
			name:           "numeric session ID",
			sessionId:      "12345",
			expectedResult: "12345-channels-default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetDefaultSSEConnectionName(tc.sessionId)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
