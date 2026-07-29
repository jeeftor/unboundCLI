package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"

	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
	"github.com/spf13/cobra"
)

var (
	statusCaddyServerIP   string
	statusCaddyServerPort int
	statusIssuesOnly      bool
	statusHostnameFilter  string
	statusCompact         bool
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show full sync status across all services including Cloudflare",
	Long: `Display a comprehensive status report for every hostname in the pipeline:
Caddy routes, DNS sync state (Unbound/AdGuard), DHCP leases, Cloudflare tunnel
rules, and any detected issues (stale entries, missing CNAMEs, wrong headers).`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, _ []string) error {
	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:     statusCaddyServerIP,
		CaddyServerPort:   statusCaddyServerPort,
		IncludeUnbound:    true,
		IncludeDNSMasq:    true,
		IncludeAdguard:    true,
		IncludeCloudflare: true,
	})
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w", err)
	}

	if !statusCompact {
		fmt.Fprint(cmd.OutOrStdout(), StyleMuted.Render("Loading data from all services…"))
	}

	// Use signal-aware context so Ctrl+C cancels pending API calls.
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	entries, report, err := status.LoadEntries(ctx, runtime.Clients, status.Options{
		CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
	})
	if err != nil {
		return fmt.Errorf("error loading data: %w", err)
	}

	if !statusCompact {
		fmt.Fprintln(cmd.OutOrStdout(), " "+SymOK)
	}

	// Apply hostname filter
	if statusHostnameFilter != "" {
		var filtered []*models.Entry
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Hostname), strings.ToLower(statusHostnameFilter)) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	out := cmd.OutOrStdout()

	if statusCompact {
		total := len(entries)
		issues := 0
		for _, e := range entries {
			if statusEntryHasIssue(e) {
				issues++
			}
		}
		if issues > 0 {
			fmt.Fprintf(out, "%s  %d entries  %s issues\n", SymWarn, total, StyleFail.Render(fmt.Sprintf("%d", issues)))
		} else {
			fmt.Fprintf(out, "%s  %d entries  all good\n", SymOK, total)
		}
		return nil
	}

	// ── Service report ─────────────────────────────────────────────────────────
	fmt.Fprintln(out)
	fmt.Fprintln(out, StyleSection.Render("── Services ─────────────────────────────────────────────────"))
	for svc, rep := range report.Services {
		icon := SymOK
		detail := fmt.Sprintf("%d entries", rep.Count)
		if rep.Error != "" {
			icon = SymFail
			detail = StyleFail.Render(rep.Error)
		} else if rep.Status == "skipped" {
			icon = StyleMuted.Render("─")
			detail = StyleMuted.Render("skipped")
		}
		fmt.Fprintf(out, "  %s  %-14s %s\n", icon, StyleBold.Render(string(svc)), detail)
	}
	fmt.Fprintln(out)

	// ── Entries table ──────────────────────────────────────────────────────────
	fmt.Fprintln(out, StyleSection.Render("── Entries ──────────────────────────────────────────────────"))
	fmt.Fprintln(out)

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
		StyleMuted.Render("HOSTNAME"),
		StyleMuted.Render("STATUS"),
		StyleMuted.Render("UNBOUND"),
		StyleMuted.Render("ADGUARD"),
		StyleMuted.Render("DHCP"),
		StyleMuted.Render("CLOUDFLARE"),
	)

	var issueEntries []*models.Entry
	for _, e := range entries {
		hasIssue := statusEntryHasIssue(e)
		if statusIssuesOnly && !hasIssue {
			continue
		}

		hostname := e.Hostname
		if hasIssue {
			hostname = StyleWarn.Render(hostname)
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			hostname,
			statusRenderSync(e.OverallStatus),
			statusRenderSvc(e.UnboundStatus.Configured, e.UnboundStatus.InSync),
			statusRenderSvc(e.AdguardStatus.Configured, e.AdguardStatus.InSync),
			statusRenderDHCP(e.DHCPStatus),
			statusRenderCF(e.CloudflareStatus),
		)

		if hasIssue {
			issueEntries = append(issueEntries, e)
		}
	}
	if err := tw.Flush(); err != nil {
		logging.Warn("Failed to flush status table", "error", err)
	}
	fmt.Fprintln(out)

	// ── Issues ─────────────────────────────────────────────────────────────────
	if len(issueEntries) > 0 {
		fmt.Fprintln(out, StyleSection.Render(fmt.Sprintf("── Issues (%d) ───────────────────────────────────────────────", len(issueEntries))))
		fmt.Fprintln(out)
		for _, e := range issueEntries {
			for _, msg := range statusIssueMessages(e) {
				fmt.Fprintf(out, "  %s  %s\n      %s\n\n",
					SymWarn,
					StyleBold.Render(e.Hostname),
					StyleMuted.Render(msg),
				)
			}
		}
		fmt.Fprintln(out, StyleInfo.Render("  TIP: visit the web UI Issues tab to fix Cloudflare routes,"))
		fmt.Fprintln(out, StyleInfo.Render("       or run 'caddy-dns-sync caddy-sync-all --dry-run' to preview DNS sync."))
		fmt.Fprintln(out)
		return exitCode(1)
	}

	fmt.Fprintf(out, "  %s  All %d entries look good.\n\n", SymOK, len(entries))
	return nil
}

func statusEntryHasIssue(e *models.Entry) bool {
	if e.OverallStatus == models.OutOfSync || e.OverallStatus == models.Stale {
		return true
	}
	if e.CloudflareStatus.Configured && !e.CloudflareStatus.HasDNSRecord {
		return true
	}
	if e.NeedsHTTPHostHeader() {
		return true
	}
	return false
}

func statusIssueMessages(e *models.Entry) []string {
	var msgs []string
	switch e.OverallStatus {
	case models.OutOfSync:
		msgs = append(msgs, "DNS sync out of date — run caddy-sync-all to fix")
	case models.Stale:
		msgs = append(msgs, "Entry in Unbound/AdGuard but no longer in Caddy (stale)")
	}
	if e.CloudflareStatus.Configured && !e.CloudflareStatus.HasDNSRecord {
		msgs = append(msgs, "Cloudflare tunnel rule exists but no CNAME DNS record — host unreachable externally")
	}
	if e.NeedsHTTPHostHeader() {
		msgs = append(msgs, "Cloudflare tunnel missing HTTP Host header — Caddy routing will break")
	}
	return msgs
}

func statusRenderSync(s models.SyncStatus) string {
	switch s {
	case models.FullyInSync:
		return StyleOK.Render("✓ synced")
	case models.PartiallyInSync:
		return StyleWarn.Render("~ partial")
	case models.OutOfSync:
		return StyleFail.Render("✗ out of sync")
	case models.CaddyOnly:
		return StyleWarn.Render("+ caddy only")
	case models.Stale:
		return StyleFail.Render("- stale")
	default:
		return StyleMuted.Render("? unknown")
	}
}

func statusRenderSvc(configured, inSync bool) string {
	if !configured {
		return StyleMuted.Render("─")
	}
	if inSync {
		return StyleOK.Render("✓")
	}
	return StyleFail.Render("✗")
}

func statusRenderDHCP(d models.DHCPStatus) string {
	if !d.Configured {
		return StyleMuted.Render("─")
	}
	if d.Type == "static" {
		return StyleOK.Render(d.IP)
	}
	return StyleWarn.Render(d.IP + " (dyn)")
}

func statusRenderCF(cf models.CloudflareStatus) string {
	if !cf.Configured {
		return StyleMuted.Render("─")
	}
	name := cf.TunnelName
	if name == "" {
		name = "tunnel"
	}
	if !cf.HasDNSRecord {
		return StyleFail.Render("✗ " + name + " (no CNAME)")
	}
	if cf.HTTPHostHeader == "" {
		return StyleWarn.Render("~ " + name + " (no host hdr)")
	}
	return StyleOK.Render("✓ " + name)
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVar(&statusCaddyServerIP, "caddy-ip", runtimeapp.DefaultCaddyServerIP, "IP address of the Caddy server")
	statusCmd.Flags().IntVar(&statusCaddyServerPort, "caddy-port", runtimeapp.DefaultCaddyServerPort, "Admin port of the Caddy server")
	statusCmd.Flags().BoolVar(&statusIssuesOnly, "issues-only", false, "Show only entries with issues")
	statusCmd.Flags().StringVar(&statusHostnameFilter, "hostname", "", "Filter by hostname (partial match)")
	statusCmd.Flags().BoolVar(&statusCompact, "compact", false, "Show one-line summary only")
}
