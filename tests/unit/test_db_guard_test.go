package unit

import (
	"testing"

	"app/config"

	"github.com/stretchr/testify/assert"
)

func TestAssertTestDatabaseURL(t *testing.T) {
	cases := []struct {
		name        string
		url         string
		allowRemote bool
		wantErr     bool
	}{
		{"local test database", "postgres://postgres@localhost:5432/gogo_test?sslmode=disable", false, false},
		{"loopback test database", "postgres://postgres@127.0.0.1:5432/gogo_test", false, false},
		{"database without _test suffix", "postgres://postgres@localhost:5432/gogo?sslmode=disable", false, true},
		{"remote host", "postgres://postgres@db.example.com:5432/gogo_test", false, true},
		{"remote host allowed explicitly", "postgres://postgres@db.example.com:5432/gogo_test", true, false},
		{"remote host allowed but wrong name", "postgres://postgres@db.example.com:5432/gogo", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.allowRemote {
				t.Setenv("ALLOW_REMOTE_TEST_DB", "1")
			} else {
				t.Setenv("ALLOW_REMOTE_TEST_DB", "")
			}

			err := config.AssertTestDatabaseURL(tc.url)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
