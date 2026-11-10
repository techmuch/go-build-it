package artifacts

import "testing"

// go-build-it's own docs/ follow the Project documents standard (FR-150).
func TestOwnDocsFollowTheStandard(t *testing.T) {
	m, err := Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range Check("../..", m) {
		t.Errorf("%s: %s", p.Kind, p.Detail)
	}
}
