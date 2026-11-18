package services

import "testing"

func TestInternal_TruncateUTF8(t *testing.T) {
	// Length <= maxBytes
	if s := truncateUTF8("hello", 10); s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}

	// maxBytes <= 3
	if s := truncateUTF8("hello", 2); s != "he" {
		t.Errorf("expected 'he', got %q", s)
	}
	if s := truncateUTF8("hello", 3); s != "hel" {
		t.Errorf("expected 'hel', got %q", s)
	}

	// maxBytes > 3 -> appends "..."
	if s := truncateUTF8("hello world", 8); s != "hello..." {
		t.Errorf("expected 'hello...', got %q", s)
	}
}
