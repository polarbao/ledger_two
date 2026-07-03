package metadata

import "testing"

func TestParseKind(t *testing.T) {
	tests := []struct {
		value string
		ok    bool
	}{
		{"categories", true},
		{"tags", true},
		{"accounts", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		_, ok := ParseKind(tt.value)
		if ok != tt.ok {
			t.Fatalf("ParseKind(%q) ok=%v, want %v", tt.value, ok, tt.ok)
		}
	}
}

func TestCanManage(t *testing.T) {
	if !CanManage("owner") {
		t.Fatalf("owner should manage metadata")
	}
	if CanManage("editor") {
		t.Fatalf("editor should not manage metadata by default")
	}
	if CanManage("viewer") {
		t.Fatalf("viewer should not manage metadata")
	}
}
