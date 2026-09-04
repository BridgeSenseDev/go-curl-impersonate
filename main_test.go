package curl

import (
	"fmt"
	"os"
	"testing"
)

// TestMain gives the process a cache directory of its own. The package writes
// its CA bundle under os.UserCacheDir once per process, and the TLS tests
// append a test certificate to that file; nothing outside this process must
// learn to trust it.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "go-curl-impersonate-test-home")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CACHE_HOME", home)
	os.Setenv("LocalAppData", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
