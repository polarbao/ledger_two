package classifier

import "testing"

func TestNormalizeTextUsesNFKCLowercaseAndStableWhitespace(t *testing.T) {
	tests := map[string]string{
		" 星河咖啡　":          "星河咖啡",
		"STAR　  CAFE":     "star cafe",
		"ＡＢＣ，测试":          "abc,测试",
		"Apple\t\n Store": "apple store",
	}
	for input, want := range tests {
		if got := NormalizeText(input); got != want {
			t.Fatalf("NormalizeText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeTextDoesNotMergeDifferentSubjects(t *testing.T) {
	store := NormalizeText("Apple Store")
	music := NormalizeText("Apple Music")
	if store == music {
		t.Fatalf("different subjects normalized to the same key %q", store)
	}
}
