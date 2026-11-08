package app

import (
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/rehearsals"
	"github.com/techmuch/go-build-it/internal/standards"
)

func historyQuery(project string) history.Query { return history.Query{ProjectID: project, Limit: 500} }
func historyQueryType(t string) history.Query   { return history.Query{Type: t, Limit: 50} }
func rehearsalOf(project, milestone string) rehearsals.Rehearsal {
	return rehearsals.Rehearsal{ProjectID: project, MilestoneID: milestone, Title: "Design tabletop",
		Steps: []rehearsals.Step{{Title: "Open the screen", Scenario: "You open it"}}}
}

func standardsRender(p string, s []standards.Standard) string { return standards.Render(p, s, "") }
