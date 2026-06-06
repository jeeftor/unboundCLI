package cmd

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewWebTokenReturnsURLSafeToken(t *testing.T) {
	token, err := newWebToken()
	if err != nil {
		t.Fatalf("newWebToken failed: %v", err)
	}
	if len(token) < 24 {
		t.Fatalf("expected token length >= 24, got %d", len(token))
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("expected raw URL-safe token, got %q", token)
	}
}

func TestServeWebPrintsBoundAddressAndServesIndex(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	var output bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveWebForTest(listener, "test-token", &output)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-errCh:
		case <-time.After(time.Second):
			t.Fatal("server did not stop after listener close")
		}
	})

	resp, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("GET web index failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(output.String(), "Web GUI listening on http://") {
		t.Fatalf("startup output missing URL: %q", output.String())
	}
	if strings.Contains(output.String(), "test-token") {
		t.Fatalf("startup output should not print inactive token: %q", output.String())
	}
}
