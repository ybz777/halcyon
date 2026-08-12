package webhook

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := "whsec_testsecretvalue1234567890"
	payload := []byte(`{"event":"test"}`)
	ts := time.Now().UTC().Unix()
	sig := SignPayload(secret, ts, payload)
	if err := VerifySignature(secret, ts, payload, sig, time.Now().UTC(), DefaultMaxSkew); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func TestRejectPrivateURL(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/hook",
		"https://localhost/hook",
		"https://10.0.0.1/hook",
		"https://169.254.169.254/latest/meta-data",
	}
	for _, raw := range cases {
		if err := ValidateDestinationURL(raw, URLPolicy{}); err == nil {
			t.Fatalf("expected rejection for %s", raw)
		}
	}
}

func TestHashSecretDeterministic(t *testing.T) {
	a := HashSecret("whsec_abc")
	b := HashSecret("whsec_abc")
	if a != b {
		t.Fatal("hash should be deterministic")
	}
}

func TestOldTimestampRejected(t *testing.T) {
	secret := "whsec_testsecretvalue1234567890"
	payload := []byte(`{}`)
	ts := time.Now().UTC().Add(-10 * time.Minute).Unix()
	sig := SignPayload(secret, ts, payload)
	if err := VerifySignature(secret, ts, payload, sig, time.Now().UTC(), DefaultMaxSkew); err == nil {
		t.Fatal("expected old timestamp rejection")
	}
}
