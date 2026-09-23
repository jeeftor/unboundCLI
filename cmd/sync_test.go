package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncDryRunModeRequiresExplicitApply(t *testing.T) {
	originalApply := syncApply
	defer func() { syncApply = originalApply }()

	command := &cobra.Command{Use: "sync"}
	command.Flags().Bool("dry-run", false, "")

	syncApply = false
	dryRun, err := syncDryRunMode(command)
	if err != nil || !dryRun {
		t.Fatalf("default mode = dryRun %t, err %v; want preview", dryRun, err)
	}

	syncApply = true
	dryRun, err = syncDryRunMode(command)
	if err != nil || dryRun {
		t.Fatalf("apply mode = dryRun %t, err %v; want execution", dryRun, err)
	}

	if err := command.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("set dry-run flag: %v", err)
	}
	if _, err := syncDryRunMode(command); err == nil {
		t.Fatal("expected --apply and --dry-run to conflict")
	}
}
