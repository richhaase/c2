package jsonx

import "testing"

func TestCompactDoesNotEscapeHTML(t *testing.T) {
	got, err := Compact(map[string]string{"body": "pace < 2:00 & HR > 150"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"body":"pace < 2:00 & HR > 150"}`
	if string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestIndentMatchesTwoSpaceStyle(t *testing.T) {
	got, err := Indent(map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": 1\n}"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNoTrailingNewline(t *testing.T) {
	got, err := Compact([]int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "[1,2]" {
		t.Fatalf("got %q", got)
	}
}
