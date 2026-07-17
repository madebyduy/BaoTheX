package telegram

import "testing"

// The gate matters more than it looks. Telegram validates an inline button's URL
// on its own servers and rejects a host it will not resolve; that rejection
// fails the entire sendMessage, so a bad button does not degrade the reply — it
// deletes it. Every false positive here turns "/start" into silence.
func TestButtonURLAllowed(t *testing.T) {
	allowed := []string{
		"https://baothex.vn",
		"https://www.baothex.vn/",
		"http://staging.baothex.vn:8080",
	}
	for _, raw := range allowed {
		if !buttonURLAllowed(raw) {
			t.Errorf("public URL rejected: %q", raw)
		}
	}

	rejected := []string{
		"",                      // unset PUBLIC_BASE_URL
		"http://localhost:3000", // the default dev address
		"https://localhost",     //
		"http://127.0.0.1:3000", //
		"http://[::1]:3000",     //
		"http://baothex.local",  // mDNS name, resolvable only on the LAN
		"http://dev-box:3000",   // bare hostname, no dot
		"ftp://baothex.vn",      // Telegram takes http(s) only
		"baothex.vn",            // no scheme
		"://nonsense",           // unparseable
	}
	for _, raw := range rejected {
		if buttonURLAllowed(raw) {
			t.Errorf("unusable URL accepted: %q", raw)
		}
	}
}

// connectButton is called unconditionally by handleStart, so it has to be safe
// on a dev machine: nil buttons make SendMessage send plain text.
func TestConnectButtonDropsDevAddress(t *testing.T) {
	dev := &Handler{baseURL: "http://localhost:3000"}
	if b := dev.connectButton(); b != nil {
		t.Fatalf("dev address produced a button that Telegram would reject: %+v", b)
	}

	prod := &Handler{baseURL: "https://baothex.vn"}
	b := prod.connectButton()
	if len(b) != 1 || len(b[0]) != 1 {
		t.Fatalf("expected one button, got %+v", b)
	}
	if want := "https://baothex.vn/cai-dat"; b[0][0].URL != want {
		t.Errorf("button URL = %q, want %q", b[0][0].URL, want)
	}
	if b[0][0].Text == "" {
		t.Error("button has no label")
	}
}
