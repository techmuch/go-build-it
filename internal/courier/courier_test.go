package courier

import "testing"

func TestParseWorkflow(t *testing.T) {
	w := ParseWorkflow("Branch per task → PR → Lens review → human approves → squash-merge → tag YYYY.MM.DD.N → publish Pages")
	if !w.RequirePR || !w.Squash || !w.HumanApproval || !w.LensReview || !w.TagReleases || !w.Pages {
		t.Fatalf("%+v", w)
	}
	w = ParseWorkflow("Commit straight to main. No pull request. Merge commits.")
	if w.RequirePR || w.Squash {
		t.Fatalf("%+v", w)
	}
	if n := ReleaseNotes("2026.11.08.1", []string{"T-1: a", "T-2: b"}); n != "## 2026.11.08.1\n\n- T-1: a\n- T-2: b\n" {
		t.Fatalf("notes %q", n)
	}
}
