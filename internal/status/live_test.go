package status

import (
	"os"
	"testing"
)

func TestLiveServiceTestsRequireExplicitOptIn(t *testing.T) {
	if os.Getenv("UNBOUNDCLI_LIVE_TESTS") != "1" {
		t.Skip("set UNBOUNDCLI_LIVE_TESTS=1 to run live service checks")
	}

	for _, name := range []string{
		"UNBOUND_CLI_BASE_URL",
		"UNBOUND_CLI_API_KEY",
		"UNBOUND_CLI_API_SECRET",
	} {
		if os.Getenv(name) == "" {
			t.Skipf("set %s for live service checks", name)
		}
	}
}
