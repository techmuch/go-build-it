package docs

import "testing"

func TestPagesAndSearch(t *testing.T) {
	ps := Pages()
	if len(ps) == 0 {
		t.Fatal("no docs embedded")
	}
	p, ok := Get(ps[0].Slug)
	if !ok || p.Title == "" || p.Body == "" {
		t.Fatalf("page %+v", p)
	}
	if hits := Search("install"); len(hits) == 0 {
		t.Fatal("search found nothing for 'install'")
	}
}
