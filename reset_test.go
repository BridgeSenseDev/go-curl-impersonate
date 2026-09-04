package curl

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// trustServer appends the test server's certificate to the CA bundle the
// package writes on first use, so that bundle alone trusts the server.
func trustServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	path, err := getEmbeddedCACertPath()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}); err != nil {
		t.Fatal(err)
	}
}

// TestResetKeepsEmbeddedCABundle: a handle verifies peers against the embedded
// bundle after Reset as it does after EasyInit. Seam: Reset, where
// curl_easy_reset clears CURLOPT_CAINFO with every other option. Only the
// embedded bundle trusts the test server, so a reset handle that fell back to
// the library's default store fails the second transfer.
func TestResetKeepsEmbeddedCABundle(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	trustServer(t, server)

	easy := EasyInit()
	defer easy.Cleanup()
	for transfer := 1; transfer <= 2; transfer++ {
		if transfer > 1 {
			easy.Reset()
		}
		easy.Setopt(OPT_URL, server.URL)
		easy.Setopt(OPT_SSL_VERIFYPEER, true)
		easy.Setopt(OPT_SSL_VERIFYHOST, true)
		easy.Setopt(OPT_WRITEFUNCTION, func([]byte, any) bool { return true })
		if err := easy.Perform(); err != nil {
			t.Fatalf("transfer %d: %v", transfer, err)
		}
	}
}
