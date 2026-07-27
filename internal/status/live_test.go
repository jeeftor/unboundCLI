package status

import (
	"os"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

func TestLiveServiceTestsRequireExplicitOptIn(t *testing.T) {
	// CADDY_DNS_SYNC_LIVE_TESTS is the preferred flag; UNBOUNDCLI_LIVE_TESTS
	// remains supported as a deprecated alias.
	if os.Getenv("CADDY_DNS_SYNC_LIVE_TESTS") != "1" && os.Getenv("UNBOUNDCLI_LIVE_TESTS") != "1" {
		t.Skip("set CADDY_DNS_SYNC_LIVE_TESTS=1 to run live service checks")
	}

	// Accept both new (CADDY_DNS_SYNC_*) and deprecated (UNBOUND_CLI_*) env vars.
	for _, pair := range [][2]string{
		{config.EnvBaseURL, config.EnvBaseURLDeprecated},
		{config.EnvAPIKey, config.EnvAPIKeyDeprecated},
		{config.EnvAPISecret, config.EnvAPISecretDeprecated},
	} {
		if os.Getenv(pair[0]) == "" && os.Getenv(pair[1]) == "" {
			t.Skipf("set %s (or deprecated %s) for live service checks", pair[0], pair[1])
		}
	}
}
