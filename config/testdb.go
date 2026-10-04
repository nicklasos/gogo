package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// testDBNameSuffix is the mandatory suffix for any database the test suite may touch,
// so a TEST_DATABASE_URL set by hand or inherited from a shell cannot point at real data.
const testDBNameSuffix = "_test"

// allowRemoteTestDBEnv opts out of the local-host requirement for CI runners where
// Postgres is reachable under a service name. It cannot bypass the name suffix check.
const allowRemoteTestDBEnv = "ALLOW_REMOTE_TEST_DB"

var localTestDBHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
	"":          true,
}

// AssertTestDatabaseURL rejects connection strings that do not clearly point at a
// throwaway test database. Tests create and truncate data unconditionally, so an
// accidental production URL is destructive rather than merely wrong.
func AssertTestDatabaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("TEST_DATABASE_URL is not a valid URL: %w", err)
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.HasSuffix(dbName, testDBNameSuffix) {
		return fmt.Errorf(
			"refusing to use database %q: TEST_DATABASE_URL must point at a database whose name ends in %q",
			dbName, testDBNameSuffix,
		)
	}

	if os.Getenv(allowRemoteTestDBEnv) == "1" {
		return nil
	}

	if !localTestDBHosts[u.Hostname()] {
		return fmt.Errorf(
			"refusing to use host %q: TEST_DATABASE_URL must point at localhost, or set %s=1 to allow a remote test database",
			u.Hostname(), allowRemoteTestDBEnv,
		)
	}

	return nil
}

// UseTestDatabase points the config at TEST_DATABASE_URL after checking it is safe.
func (c *Config) UseTestDatabase() error {
	if c.TestDatabaseURL == "" {
		return fmt.Errorf("TEST_DATABASE_URL environment variable is required")
	}
	if err := AssertTestDatabaseURL(c.TestDatabaseURL); err != nil {
		return err
	}
	c.DatabaseURL = c.TestDatabaseURL
	return nil
}
