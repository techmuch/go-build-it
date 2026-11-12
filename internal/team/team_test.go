package team

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

func setup(t *testing.T) (*Service, context.Context) {
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db, history.New(db)), access.WithActor(context.Background(), access.Actor{Name: "david", Kind: "human", Origin: access.Local})
}

func TestStarterArchetypesAndTeam(t *testing.T) {
	s, ctx := setup(t)
	as, err := s.Archetypes(ctx)
	if err != nil || len(as) != len(BuiltinRoles) {
		t.Fatalf("archetypes %d %v", len(as), err)
	}
	if as[0].Role != "pm" || !as[0].Builtin || as[0].Rev != 1 {
		t.Fatalf("first archetype %+v", as[0])
	}
	ms, err := s.EnsureStarterTeam(ctx, "inboxql")
	if err != nil || len(ms) != len(StarterTeam) {
		t.Fatalf("starter team %d %v", len(ms), err)
	}
	dev, err := s.Member(ctx, "forge@inboxql")
	if err != nil || dev.Harness != "claude" || dev.Role != "developer" || !strings.Contains(dev.Instructions, "tests first") || dev.Tuned() {
		t.Fatalf("developer %+v %v", dev, err)
	}
	if again, _ := s.EnsureStarterTeam(ctx, "inboxql"); len(again) != len(StarterTeam) {
		t.Fatal("starter team must not be added twice")
	}
	if devs, _ := s.ByRole(ctx, "developer", "inboxql"); len(devs) != 1 {
		t.Fatalf("by role %d", len(devs))
	}
	if devs, _ := s.ByRole(ctx, "developer", "other"); len(devs) != 0 {
		t.Fatal("members are per project")
	}
}

// Starter instructions nobody edited are upgraded when go-build-it ships new ones; edited
// ones, and the owner's harness choice, are left alone. Members follow through the change review.
func TestStarterInstructionsUpgrade(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db, history.New(db))
	ctx := access.WithActor(context.Background(), access.Actor{Name: "david", Kind: "human", Origin: access.Local})
	// Put the install back to what an earlier release shipped, then let the owner edit two.
	for id, old := range starterV1 {
		if _, err := db.Exec(`UPDATE archetypes SET instructions=? WHERE id=?`, old, id); err != nil {
			t.Fatal(err)
		}
	}
	rev, _ := s.Archetype(ctx, "reviewer")
	rev.Harness = "antigravity" // the owner changed only the harness
	if _, err := s.SaveArchetype(ctx, rev); err != nil {
		t.Fatal(err)
	}
	arch, _ := s.Archetype(ctx, "architect")
	arch.Instructions = "## Mine\n- My own rules.\n"
	if _, err := s.SaveArchetype(ctx, arch); err != nil {
		t.Fatal(err)
	}
	m, _ := s.AddFromArchetype(ctx, "p1", "developer", "Forge")

	s = New(db, history.New(db)) // the next start of the upgraded binary
	dev, _ := s.Archetype(ctx, "developer")
	if dev.Rev != 2 || !strings.Contains(dev.Instructions, "Never weaken, skip or delete a test") {
		t.Fatalf("developer not upgraded: rev %d\n%s", dev.Rev, dev.Instructions)
	}
	if r, _ := s.Archetype(ctx, "reviewer"); r.Rev != 3 || r.Harness != "antigravity" || !strings.Contains(r.Instructions, "independent second pair of eyes") {
		t.Fatalf("reviewer: rev %d harness %s", r.Rev, r.Harness)
	}
	if a, _ := s.Archetype(ctx, "architect"); a.Rev != 2 || !strings.Contains(a.Instructions, "My own rules") {
		t.Fatalf("an edited archetype was changed: rev %d\n%s", a.Rev, a.Instructions)
	}
	if again := New(db, history.New(db)); again != nil {
		if d, _ := again.Archetype(ctx, "developer"); d.Rev != 2 {
			t.Fatalf("upgrade ran twice: rev %d", d.Rev)
		}
	}
	cs, _ := s.Changes(ctx, "developer")
	if len(cs) != 1 || cs[0].Member.ID != m.ID || !cs[0].CanApplyClean {
		t.Fatalf("the member should be offered the new revision: %+v", cs)
	}
	var roleDesc string
	db.QueryRow(`SELECT description FROM roles WHERE id='qa'`).Scan(&roleDesc)
	if !strings.Contains(roleDesc, "risk-based") {
		t.Fatalf("built-in role description not refreshed: %s", roleDesc)
	}
}

func TestTuningChangesPromote(t *testing.T) {
	s, ctx := setup(t)
	forge, err := s.AddFromArchetype(ctx, "p1", "developer", "Forge")
	if err != nil {
		t.Fatal(err)
	}
	spark, _ := s.AddFromArchetype(ctx, "p2", "developer", "Spark")
	tuning, model := "- Ask before adding a dependency.\n- Go only.", "opus"
	forge, err = s.Tune(ctx, forge.ID, Tuning{Tuning: &tuning, Model: &model})
	if err != nil {
		t.Fatal(err)
	}
	if !forge.Tuned() || forge.Model != "opus" || !strings.Contains(forge.Instructions, "## For this project") || !strings.Contains(forge.Instructions, "Go only") {
		t.Fatalf("tuned %+v", forge)
	}
	repl, rtext := true, "Totally different."
	if _, err := s.Tune(ctx, spark.ID, Tuning{ReplaceInstructions: &repl, Tuning: &rtext}); err != nil {
		t.Fatal(err)
	}
	// New archetype revision: instructions and model change.
	a, _ := s.Archetype(ctx, "developer")
	a.Instructions += "- Keep functions small.\n"
	a.Model = "sonnet"
	a2, err := s.SaveArchetype(ctx, a)
	if err != nil || a2.Rev != 2 || a2.UsedBy != 2 {
		t.Fatalf("save %+v %v", a2, err)
	}
	// Members stay on rev 1 until the change is applied (FR-12b: never silently).
	if m, _ := s.Member(ctx, forge.ID); m.ArchetypeRev != 1 || m.ArchetypeLatest != 2 || strings.Contains(m.Instructions, "functions small") {
		t.Fatalf("member moved silently: %+v", m)
	}
	cs, err := s.Changes(ctx, "developer")
	if err != nil || len(cs) != 2 {
		t.Fatalf("changes %d %v", len(cs), err)
	}
	byID := map[string]Change{}
	for _, c := range cs {
		byID[c.Member.ID] = c
	}
	if c := byID[forge.ID]; c.CanApplyClean || strings.Join(c.Overrides, ",") != "model" {
		t.Fatalf("forge overrides %+v", c)
	}
	if c := byID[spark.ID]; c.CanApplyClean || strings.Join(c.Overrides, ",") != "instructions" {
		t.Fatalf("spark overrides %+v", c)
	}
	// Forge overrides the model, so it is not clean either; apply it explicitly, keeping tuning.
	m, err := s.Apply(ctx, forge.ID)
	if err != nil || m.ArchetypeRev != 2 || !strings.Contains(m.Instructions, "functions small") || !strings.Contains(m.Instructions, "Go only") || m.Model != "opus" {
		t.Fatalf("apply %+v %v", m, err)
	}
	if applied, _ := s.ApplyClean(ctx, "developer"); len(applied) != 0 {
		t.Fatalf("spark overrides instructions and must not be applied in bulk: %d", len(applied))
	}
	// Promote a tuning line back to the archetype.
	ar, m, err := s.Promote(ctx, forge.ID, "- Ask before adding a dependency.")
	if err != nil || ar.Rev != 3 || !strings.Contains(ar.Instructions, "Ask before adding a dependency") {
		t.Fatalf("promote %+v %v", ar, err)
	}
	if strings.Contains(m.Tuning, "dependency") || !strings.Contains(m.Tuning, "Go only") || m.ArchetypeRev != 3 {
		t.Fatalf("member after promote %+v", m)
	}
	if err := s.RemoveArchetype(ctx, "developer"); err == nil {
		t.Fatal("an archetype in use must not be removable")
	}
	// Archetype IDs resolve to a virtual member for global work.
	if v, err := s.Member(ctx, "reviewer"); err != nil || v.Role != "reviewer" || v.ProjectID != "" {
		t.Fatalf("virtual member %+v %v", v, err)
	}
	if _, err := s.Tune(ctx, "reviewer", Tuning{Tuning: &tuning}); err == nil {
		t.Fatal("archetypes are edited in the library, not tuned")
	}
}

func TestHumansPerProject(t *testing.T) {
	s, ctx := setup(t)
	m, err := s.AddHuman(ctx, "p1", "maria", "maria@github", "reviewer")
	if err != nil || m.ID != "maria@p1" || m.Kind != "human" {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := s.AddHuman(ctx, "p2", "maria", "maria@github", "pm"); err != nil {
		t.Fatal(err)
	}
	all, _ := s.Members(ctx, "")
	if len(all) != 2 {
		t.Fatalf("maria holds a role in two projects: %d", len(all))
	}
}
