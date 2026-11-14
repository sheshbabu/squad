package files

import (
	"squad/commons/testutil"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"notes.md", "Packing list.md", "2026-trip_plan.v2.md"}
	invalid := []string{"../secret.md", "notes.txt", ".hidden.md", "a/b.md", "", "notes.md/"}

	for _, name := range valid {
		if ValidateName(name) != nil {
			t.Errorf("expected %q to be valid", name)
		}
	}
	for _, name := range invalid {
		if ValidateName(name) == nil {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}

func TestEditFileNeedsExactlyOneMatch(t *testing.T) {
	testutil.SetupDB(t)

	if err := WriteFile(1, "list.md", "- eggs\n- milk\n- eggs\n"); err != nil {
		t.Fatal(err)
	}

	if EditFile(1, "list.md", "bread", "toast") == nil {
		t.Error("expected a missing match to fail")
	}
	if EditFile(1, "list.md", "eggs", "ham") == nil {
		t.Error("expected an ambiguous match to fail")
	}
	if err := EditFile(1, "list.md", "milk", "oat milk"); err != nil {
		t.Fatal(err)
	}

	fileContent, _ := ReadFile(1, "list.md")
	if fileContent.Content != "- eggs\n- oat milk\n- eggs\n" {
		t.Fatalf("unexpected content %q", fileContent.Content)
	}
}
