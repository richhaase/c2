package jsonx

import "testing"

func TestCompactDoesNotEscapeHTML(t *testing.T) {
	payload := struct {
		Body string `json:"body"`
	}{
		Body: "pace < 2:00 & HR > 150",
	}
	got, err := Compact(payload)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"body":"pace < 2:00 & HR > 150"}`
	if string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestIndentMatchesTwoSpaceStyle(t *testing.T) {
	payload := struct {
		A int `json:"a"`
	}{
		A: 1,
	}
	got, err := Indent(payload)
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
