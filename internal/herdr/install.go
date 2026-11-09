package herdr

import "runtime"

// InstallHint says how to install herdr on this platform (FR-07).
func InstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install herdr"
	case "windows":
		return `powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"   (Windows support is beta)`
	default:
		return "curl -fsSL https://herdr.dev/install.sh | sh"
	}
}
