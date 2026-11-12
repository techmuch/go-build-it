// Package doctor proves that go-build-it can run its agents: herdr is installed and
// compatible, the gbi session can open a pane and run a command, and each harness starts
// inside herdr (FR-07a–FR-07e, ADR-012). Every result explains itself to a person and to
// an AI agent: a stable id, evidence, and a fix that is either automatic or for a human.
//
// It runs inside the gbi binary and does not need `gbi serve`.
package doctor

// Check describes one doctor check. The troubleshooting page (docs/user/20-troubleshooting.md)
// has one section per id, and a test keeps the two in step.
type Check struct {
	ID    string
	Tier  Tier
	Title string // what is checked, in plain words; %s is the harness label for per-harness checks
}

// Checks lists every check in the order they run.
var Checks = []Check{
	{"herdr.installed", Static, "herdr is installed"},
	{"herdr.version", Static, "herdr version is supported"},
	{"herdr.api", Static, "herdr has every API method go-build-it uses"},
	{"harness.any", Static, "At least one agent harness is installed"},
	{"harness.installed", Static, "%s is installed"},
	{"service.installed", Static, "The login service is installed"},
	{"tailscale.serve", Static, "go-build-it is reachable and private on the tailnet"},
	{"harness.on_service_path", Static, "%s is on the login service's PATH"},
	{"session.running", Live, "The gbi herdr session is running"},
	{"pane.roundtrip", Live, "The gbi session can open a pane and run a command"},
	{"harness.on_pane_path", Live, "%s is found inside a herdr pane"},
	{"integration.current", Live, "herdr's %s integration is current"},
	{"harness.starts", HarnessTier, "%s starts in herdr"},
	{"harness.answers", Prompt, "%s answers a prompt"},
}

// CheckByID returns the check with id.
func CheckByID(id string) (Check, bool) {
	for _, c := range Checks {
		if c.ID == id {
			return c, true
		}
	}
	return Check{}, false
}

// DocsURL is the troubleshooting link for id, inside the app.
func DocsURL(id string) string { return "/docs/troubleshooting#" + id }
