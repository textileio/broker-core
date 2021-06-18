package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRunningBytesLimit(t *testing.T) {
	errorCases := []string{
		"",
		"/",
		"5/1y",
	}
	validCases := []string{
		"5/1m",
		"5 PiB/ 128h",
		"0 tib /128h",
	}
	for _, s := range errorCases {
		_, err := parseRunningBytesLimit(s)
		require.Error(t, err)
	}

	for _, s := range validCases {
		_, err := parseRunningBytesLimit(s)
		require.NoError(t, err)
	}
}
