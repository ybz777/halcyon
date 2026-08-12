package auth

import "testing"

func FuzzHS256Validator_ParseToken(f *testing.F) {
	f.Add("")
	f.Add("Bearer not-a-jwt")
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature")
	v, err := NewHS256Validator(HS256Config{
		Secret: []byte("fuzz-test-secret-at-least-32-bytes"),
		Issuer: "auth",
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 4096 {
			t.Skip("bounded input")
		}
		_, _ = v.ValidateToken(raw)
	})
}
