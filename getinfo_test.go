package curl

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// tinySink keeps the padding allocations reachable; only escaping objects go
// through the tiny allocator.
var tinySink []*byte

// TestGetinfoWritesOnlyItsOutParameter: a Getinfo out-parameter has the width
// of the C type libcurl writes through it. Seam: the CURLINFO_LONG case, whose
// Go variable was an int32 while easy_getinfo_long_helper takes a long*, so on
// LP64 (linux, darwin) libcurl zeroed the 4 bytes after it. Those bytes belong
// to the tiny allocator's next object: the sentinels are 14-byte noscan
// strings, one per 16-byte block, alternating live and garbage, so after a GC
// every reused block has a live sentinel above it and a 4-byte overrun from
// offset 12 zeroes that sentinel's first bytes.
func TestGetinfoWritesOnlyItsOutParameter(t *testing.T) {
	const body = "short and stout"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(body))
	}))
	defer server.Close()

	easy := EasyInit()
	defer easy.Cleanup()
	easy.Setopt(OPT_URL, server.URL)
	easy.Setopt(OPT_WRITEFUNCTION, func([]byte, any) bool { return true })
	if err := easy.Perform(); err != nil {
		t.Fatal(err)
	}

	const sentinel = "upstream_error"
	live := make([]string, 0, 1<<15)
	var garbage string
	for range cap(live) {
		live = append(live, strings.ToLower("UPSTREAM_ERROR"))
		garbage = strings.ToLower("UPSTREAM_ERROR")
	}
	_ = garbage
	runtime.GC()

	for i := range 4096 {
		// Walk the tiny offset one byte per iteration so the out-parameter
		// visits every slot of a block, offset 12 included.
		for range i % 16 {
			tinySink = append(tinySink, new(byte))
		}
		code, err := easy.Getinfo(INFO_RESPONSE_CODE)
		if err != nil {
			t.Fatal(err)
		}
		if code != int64(http.StatusTeapot) {
			t.Fatalf("iteration %d: INFO_RESPONSE_CODE = %v, want %d", i, code, http.StatusTeapot)
		}
		size, err := easy.Getinfo(INFO_SIZE_DOWNLOAD)
		if err != nil {
			t.Fatal(err)
		}
		if size != float64(len(body)) {
			t.Fatalf("iteration %d: INFO_SIZE_DOWNLOAD = %v, want %d", i, size, len(body))
		}
	}

	corrupted := 0
	first := ""
	for _, value := range live {
		if value != sentinel {
			if corrupted == 0 {
				first = value
			}
			corrupted++
		}
	}
	if corrupted != 0 {
		t.Fatalf("%d of %d sentinel strings were overwritten; first is %q", corrupted, len(live), first)
	}
}
