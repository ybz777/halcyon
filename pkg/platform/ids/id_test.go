package ids

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	id, err := Generate("usr")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.HasPrefix(id, "usr_") {
		t.Errorf("id = %q, want usr_ prefix", id)
	}

	id2, err := Generate("usr")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if id == id2 {
		t.Error("expected unique IDs")
	}
}

func TestGenerateEmptyPrefix(t *testing.T) {
	_, err := Generate("")
	if err == nil {
		t.Fatal("expected error for empty prefix")
	}
}

func TestGenerateWithTrailingUnderscore(t *testing.T) {
	id, err := Generate("org_")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.HasPrefix(id, "org_") {
		t.Errorf("id = %q, want org_ prefix", id)
	}
}
