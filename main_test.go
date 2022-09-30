package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_denylist(t *testing.T) {
	testCases := []struct {
		Name     string
		Host     string
		Expected bool
	}{
		{
			Name:     "Valid host",
			Host:     "1.2.3.4",
			Expected: false,
		},
		{
			Name:     "Invalid host localhost",
			Host:     "localhost",
			Expected: true,
		},
		{
			Name:     "Invalid host 0.0.0.0",
			Host:     "0.0.0.0",
			Expected: true,
		},
		{
			Name:     "Invalid host 127.0.0.1",
			Host:     "127.0.0.1",
			Expected: true,
		},
		{
			Name:     "Invalid host empty",
			Host:     "",
			Expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.Expected, denylist(tc.Host))
		})
	}
}
