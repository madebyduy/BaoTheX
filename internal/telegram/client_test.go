package telegram

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// A structurally valid token that has never existed. The first draft of this
// test used the deployment's real token, copied straight out of the log line the
// test exists to prevent — which would have committed a live credential to the
// repository in the name of stopping a leak. Test fixtures that look like
// secrets must be invented, never borrowed.
const fakeToken = "1234567890:AAaaBBbbCCccDDddEEeeFFffGGgg-HHhhIIii"

// A transport failure must not put the bot token in the returned error.
//
// This is not hypothetical. A TLS handshake timeout during polling logged the
// whole request URL, and Telegram puts the token in the path, so the line read:
//
//	err="Post \"https://api.telegram.org/bot<TOKEN>/deleteWebhook\": TLS handshake timeout"
//
// That is a working credential written to a log by a passing network blip, kept
// for as long as logs are kept, and readable by anyone who can read them.
func TestTransportErrorDoesNotLeakToken(t *testing.T) {
	c := NewClient(fakeToken)
	// A port nothing listens on: Do fails inside net/http and returns *url.Error,
	// which formats the URL it was given.
	c.http = &http.Client{Timeout: 300 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:1/bot"+fakeToken+"/deleteWebhook", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.do(req)
	if err == nil {
		t.Fatal("expected the request to fail")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("bot token leaked into the error: %v", err)
	}
	if !strings.Contains(err.Error(), "<token-redacted>") {
		t.Errorf("token was neither present nor marked redacted: %v", err)
	}
}

// Redaction must not depend on a token being configured: an unconfigured client
// still makes requests that fail, and the error must survive unmangled.
func TestTransportErrorSurvivesWithoutToken(t *testing.T) {
	c := NewClient("")
	c.http = &http.Client{Timeout: 300 * time.Millisecond}

	req, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:1/", nil)
	if _, err := c.do(req); err == nil {
		t.Fatal("expected the request to fail")
	}
}
