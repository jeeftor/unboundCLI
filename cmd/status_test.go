package cmd

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
)

func TestStatusEntryHasIssueIncludesPartialAndCaddyOnly(t *testing.T) {
	for _, state := range []models.SyncStatus{models.PartiallyInSync, models.OutOfSync, models.CaddyOnly, models.Stale} {
		if !statusEntryHasIssue(&models.Entry{OverallStatus: state}) {
			t.Fatalf("status %v should be an issue", state)
		}
	}
	if statusEntryHasIssue(&models.Entry{OverallStatus: models.FullyInSync}) {
		t.Fatal("fully synced entry should not be an issue")
	}
}

func TestStatusReportHasFailures(t *testing.T) {
	if !statusReportHasFailures(status.LoadReport{Services: map[status.ServiceName]status.ServiceReport{
		status.ServiceCaddy: {Status: status.ServiceFailed, Error: "connection refused"},
	}}) {
		t.Fatal("failed service report should be an issue")
	}
	if statusReportHasFailures(status.LoadReport{Services: map[status.ServiceName]status.ServiceReport{
		status.ServiceCaddy: {Status: status.ServiceLoaded},
	}}) {
		t.Fatal("loaded service report should not be an issue")
	}
}
