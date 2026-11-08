package agent

import "testing"

func TestDiffApply(t *testing.T) {
	a := "# Doc\nline one\nline two\nline three"
	b := "# Doc\nline one\nline 2\nline three\nline four"
	h := Diff(a, b)
	if len(h) != 2 {
		t.Fatalf("hunks %+v", h)
	}
	if got := Apply(a, h, []int{0, 1}); got != b {
		t.Fatalf("apply all:\n%s", got)
	}
	if got := Apply(a, h, nil); got != a {
		t.Fatalf("apply none:\n%s", got)
	}
	if got := Apply(a, h, []int{1}); got != "# Doc\nline one\nline two\nline three\nline four" {
		t.Fatalf("apply second only:\n%s", got)
	}
}
