package search

import "testing"

func TestBuildMatchQuery(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"passport":          `"passport"*`,
		`  where "is" it? `: `"where" """is""" "it?"*`,
	}

	for input, expected := range cases {
		if actual := buildMatchQuery(input); actual != expected {
			t.Errorf("%q: expected %s, got %s", input, expected, actual)
		}
	}
}
