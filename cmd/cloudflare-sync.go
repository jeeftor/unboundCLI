package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/jeeftor/unboundCLI/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfDryRun             bool
	cfTunnelID           string
	cfAPIToken           string
	cfAccountID          string
	cfEntryDescription   string
	cfLegacyDescriptions []string
	cfForce              bool
)

// Environment variable names for Cloudflare
const (
	EnvCfAPIToken  = "CLOUDFLARE_API_TOKEN"
	EnvCfAccountID = "CLOUDFLARE_ACCOUNT_ID"
	EnvCfTunnelID  = "CLOUDFLARE_TUNNEL_ID"
)

// CloudflareSyncOptions contains options for the Cloudflare sync operation
type CloudflareSyncOptions struct {
	DryRun             bool
	TunnelID           string
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool
}

// SyncResult contains the result of the Cloudflare sync operation
type SyncResult struct {
	HostnameMap    map[string]string
	ToAdd          []string
	ToUpdate       []string
	ToUpdateDesc   []string
	ToRemove       []string
	ChangesApplied bool
	SyncOverrides  map[string]api.DNSOverride
	OtherOverrides map[string]api.DNSOverride
	ExistingCount  int
}

// SyncCloudflareWithUnbound synchronizes DNS entries between Cloudflare tunnel and Unbound
func SyncCloudflareWithUnbound(
	unboundClient *api.Client,
	cfClient *api.CloudflareClient,
	options CloudflareSyncOptions,
) (*SyncResult, error) {
	// Fetch hostname map from Cloudflare tunnel
	hostnameMap, err := cfClient.GetTunnelHostnames()
	if err != nil {
		logging.Error("Error fetching Cloudflare tunnel hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Cloudflare tunnel hostnames: %w", err)
	}

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Cloudflare tunnel config")
		return &SyncResult{HostnameMap: hostnameMap}, nil
	}

	// Get existing overrides from Unbound
	existingOverrides, err := unboundClient.GetOverrides()
	if err != nil {
		logging.Error("Error fetching overrides", "error", err)
		return nil, fmt.Errorf("error fetching overrides: %w", err)
	}

	// Organize overrides for easier processing
	syncCreatedOverrides := make(map[string]api.DNSOverride)
	otherOverrides := make(map[string]api.DNSOverride)

	for _, override := range existingOverrides {
		key := fmt.Sprintf("%s.%s", override.Host, override.Domain)
		if override.Description == options.EntryDescription ||
			isLegacyDescription(override.Description, options.LegacyDescriptions) {
			syncCreatedOverrides[key] = override
		} else {
			otherOverrides[key] = override
		}
	}

	var toAdd, toUpdate, toUpdateDesc []string
	for hostname, serverIP := range hostnameMap {
		if !strings.Contains(hostname, ".") {
			continue
		}
		parts := strings.SplitN(hostname, ".", 2)
		if len(parts) != 2 {
			logging.Warn("Skipping invalid hostname", "hostname", hostname)
			continue
		}
		override, existsInSync := syncCreatedOverrides[hostname]
		_, existsOther := otherOverrides[hostname]
		if !existsInSync && !existsOther {
			toAdd = append(toAdd, hostname)
		} else if !existsInSync && existsOther {
			if options.Verbose {
				logging.Info("Hostname already exists (not created by sync)", "hostname", hostname)
			}
		} else if existsInSync {
			needsUpdate := false
			needsDescUpdate := false
			if override.Server != serverIP {
				needsUpdate = true
			}
			if override.Description != options.EntryDescription {
				needsDescUpdate = true
			}
			if needsUpdate {
				toUpdate = append(toUpdate, hostname)
			} else if needsDescUpdate {
				toUpdateDesc = append(toUpdateDesc, hostname)
			}
		}
	}

	var toRemove []string
	for hostname := range syncCreatedOverrides {
		if _, exists := hostnameMap[hostname]; !exists {
			toRemove = append(toRemove, hostname)
		}
	}

	changesApplied := false
	if !options.DryRun {
		changesApplied = applyCloudflareChanges(
			unboundClient,
			options,
			hostnameMap,
			syncCreatedOverrides,
			toAdd,
			toUpdate,
			toUpdateDesc,
			toRemove,
		)
	}

	return &SyncResult{
		HostnameMap:    hostnameMap,
		ToAdd:          toAdd,
		ToUpdate:       toUpdate,
		ToUpdateDesc:   toUpdateDesc,
		ToRemove:       toRemove,
		ChangesApplied: changesApplied,
		SyncOverrides:  syncCreatedOverrides,
		OtherOverrides: otherOverrides,
		ExistingCount:  len(existingOverrides),
	}, nil
}

func isLegacyDescription(desc string, legacy []string) bool {
	for _, v := range legacy {
		if desc == v {
			return true
		}
	}
	return false
}

func applyCloudflareChanges(
	client *api.Client,
	options CloudflareSyncOptions,
	hostnameMap map[string]string,
	syncCreatedOverrides map[string]api.DNSOverride,
	toAdd, toUpdate, toUpdateDesc, toRemove []string,
) bool {
	changesApplied := false
	// Add new entries
	for _, hostname := range toAdd {
		parts := strings.SplitN(hostname, ".", 2)
		host, domain := parts[0], parts[1]
		serverIP := hostnameMap[hostname]
		logging.Info("Adding DNS override", "host", host, "domain", domain, "ip", serverIP)
		override := api.DNSOverride{
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      serverIP,
			Description: options.EntryDescription,
		}
		_, err := client.AddOverride(override)
		if err != nil {
			logging.Error(
				"Failed to add DNS override",
				"error",
				err,
				"host",
				host,
				"domain",
				domain,
			)
			continue
		}
		changesApplied = true
	}
	// Update existing entries (IP changes)
	for _, hostname := range toUpdate {
		override := syncCreatedOverrides[hostname]
		serverIP := hostnameMap[hostname]
		logging.Info("Updating DNS override IP",
			"host", override.Host,
			"domain", override.Domain,
			"old_ip", override.Server,
			"new_ip", serverIP)
		override.Server = serverIP
		err := client.UpdateOverride(override)
		if err != nil {
			logging.Error(
				"Failed to update DNS override IP",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}
		changesApplied = true
	}
	// Update description
	for _, hostname := range toUpdateDesc {
		override := syncCreatedOverrides[hostname]
		logging.Info("Updating DNS override description",
			"host", override.Host,
			"domain", override.Domain,
			"old_desc", override.Description,
			"new_desc", options.EntryDescription)
		override.Description = options.EntryDescription
		err := client.UpdateOverride(override)
		if err != nil {
			logging.Error(
				"Failed to update DNS override description",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}
		changesApplied = true
	}
	// Remove entries
	for _, hostname := range toRemove {
		override := syncCreatedOverrides[hostname]
		logging.Info("Removing DNS override",
			"host", override.Host,
			"domain", override.Domain)
		err := client.DeleteOverride(override.UUID)
		if err != nil {
			logging.Error(
				"Failed to remove DNS override",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}
		changesApplied = true
	}
	return changesApplied
}

// CloudflareUI provides Cloudflare-specific UI rendering functions
type CloudflareUI struct {
	Styles tui.StyleConfig
}

func NewCloudflareUI() *CloudflareUI {
	return &CloudflareUI{Styles: tui.DefaultStyles()}
}

func (ui *CloudflareUI) RenderHeader() string {
	return ui.Styles.Header.Render("☁️ Cloudflare Tunnel Sync ☁️") + "\n"
}

func (ui *CloudflareUI) RenderInfo(message string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" ℹ️ %s ", message))
}

func (ui *CloudflareUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

func (ui *CloudflareUI) RenderTunnelsList(tunnels []api.CloudflareTunnel) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Section.Render(" 🚇 Available Cloudflare Tunnels 🚇 "))
	sb.WriteString("\n\n")

	if len(tunnels) == 0 {
		sb.WriteString(ui.Styles.Warning.Render("  No tunnels found for this account"))
		sb.WriteString("\n")
		return sb.String()
	}

	for i, tunnel := range tunnels {
		sb.WriteString(fmt.Sprintf("  %d. ", i+1))
		sb.WriteString(ui.Styles.Hostname.Render(tunnel.Name))
		sb.WriteString("\n")

		sb.WriteString("     ID: ")
		sb.WriteString(ui.Styles.Dimmed.Render(tunnel.ID))
		sb.WriteString("\n")

		sb.WriteString("     Created: ")
		sb.WriteString(formatTime(tunnel.CreatedAt))
		sb.WriteString("\n")

		if len(tunnel.Connections) > 0 {
			sb.WriteString("     Status: ")
			activeCount := 0
			for _, conn := range tunnel.Connections {
				if conn.Status == "active" {
					activeCount++
				}
			}
			if activeCount > 0 {
				sb.WriteString(ui.Styles.Success.Render(fmt.Sprintf("%d active connections", activeCount)))
			} else {
				sb.WriteString(ui.Styles.Warning.Render("No active connections"))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ui *CloudflareUI) RenderTunnelInfo(tunnelID string, tunnelName string) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Section.Render(" 🚇 Cloudflare Tunnel Information 🚇 "))
	sb.WriteString("\n\n")

	sb.WriteString("  Tunnel ID: ")
	sb.WriteString(ui.Styles.Dimmed.Render(tunnelID))
	sb.WriteString("\n")

	if tunnelName != "" {
		sb.WriteString("  Tunnel Name: ")
		sb.WriteString(ui.Styles.Hostname.Render(tunnelName))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ui *CloudflareUI) RenderNoTunnelID() string {
	var sb strings.Builder
	sb.WriteString(ui.Styles.Warning.Render(" ⚠️ No tunnel ID provided "))
	sb.WriteString("\n\n")
	sb.WriteString("Please provide a tunnel ID using the --tunnel-id flag, the CLOUDFLARE_TUNNEL_ID environment variable, or use one of the tunnels listed above.")
	sb.WriteString("\n\n")
	sb.WriteString("Example: ")
	sb.WriteString(ui.Styles.Dimmed.Render("unboundCLI cloudflare-sync --tunnel-id=<tunnel-id>"))
	sb.WriteString("\n")
	sb.WriteString("Or set the environment variable: ")
	sb.WriteString(ui.Styles.Dimmed.Render("export CLOUDFLARE_TUNNEL_ID=<tunnel-id>"))
	sb.WriteString("\n")
	return sb.String()
}

func (ui *CloudflareUI) RenderFetchingMessage() string {
	return ui.Styles.Info.Render(" 📡 Fetching Cloudflare tunnel configuration... ")
}

func (ui *CloudflareUI) RenderSourceSection() string {
	return ui.Styles.Section.Render(" ☁️ Cloudflare Tunnel Routes ☁️ ")
}

func (ui *CloudflareUI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" ⚠️ %s ", message))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("Jan 02, 2006 15:04:05")
}

// SyncUI handles the UI rendering for the sync operation
type SyncUI struct {
	Styles tui.StyleConfig
}

func NewSyncUI() *SyncUI {
	return &SyncUI{Styles: tui.DefaultStyles()}
}

func (ui *SyncUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

func (ui *SyncUI) RenderHostnameCount(count int) string {
	return fmt.Sprintf("Hostnames: %d", count)
}

func (ui *SyncUI) RenderHostnameList(hostnames []string) string {
	return strings.Join(hostnames, ", ")
}

func (ui *SyncUI) RenderSummary(result *SyncResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 📊 SUMMARY OF CHANGES 📊 "))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToAdd))),
		ui.Styles.Add.Render(" ✨ Entries to add")))

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToUpdate))),
		ui.Styles.Update.Render(" 🔄 Entries to update IP")))

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToUpdateDesc))),
		ui.Styles.Update.Render(" 📝 Entries to update description")))

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToRemove))),
		ui.Styles.Remove.Render(" 🗑️ Entries to remove")))

	return sb.String()
}

func (ui *SyncUI) RenderChanges(result *SyncResult, entryDescription string) string {
	var sb strings.Builder
	// Add new entries
	if len(result.ToAdd) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" ✨ Adding new entries: "))
		sb.WriteString("\n")
		for _, hostname := range result.ToAdd {
			parts := strings.SplitN(hostname, ".", 2)
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Add.Render("ADD "))
			sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", parts[0], parts[1])))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
			sb.WriteString("\n")
		}
	}
	// Update existing entries (IP changes)
	if len(result.ToUpdate) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🔄 Updating entries (IP address): "))
		sb.WriteString("\n")
		for _, hostname := range result.ToUpdate {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Update.Render("UPDATE "))
			sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
			sb.WriteString(": ")
			sb.WriteString(ui.Styles.IP.Render(override.Server))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
			sb.WriteString("\n")
		}
	}
	// Update description
	if len(result.ToUpdateDesc) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 📝 Updating entries (description): "))
		sb.WriteString("\n")
		for _, hostname := range result.ToUpdateDesc {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Update.Render("UPDATE "))
			sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
			sb.WriteString(": ")
			sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", override.Description)))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", entryDescription)))
			sb.WriteString("\n")
		}
	}
	// Remove entries
	if len(result.ToRemove) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🗑️ Removing entries: "))
		sb.WriteString("\n")
		for _, hostname := range result.ToRemove {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Remove.Render("REMOVE "))
			sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(override.Server))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (ui *SyncUI) RenderDryRunOutput(result *SyncResult, entryDescription string) string {
	var sb strings.Builder
	sb.WriteString(ui.RenderDryRun())
	sb.WriteString(ui.RenderAddEntries(result))
	sb.WriteString(ui.RenderUpdateEntries(result))
	sb.WriteString(ui.RenderUpdateDescEntries(result, entryDescription))
	sb.WriteString(ui.RenderRemoveEntries(result))
	return sb.String()
}

func (ui *SyncUI) RenderDryRun() string {
	return "\n" + ui.Styles.DryRun.Render(" 🧪 DRY RUN - No changes will be made 🧪 ")
}

func (ui *SyncUI) RenderAddEntries(result *SyncResult) string {
	if len(result.ToAdd) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" ✨ Entries that would be added: "))
	sb.WriteString("\n")
	for _, hostname := range result.ToAdd {
		parts := strings.SplitN(hostname, ".", 2)
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Add.Render("ADD "))
		sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", parts[0], parts[1])))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ui *SyncUI) RenderUpdateEntries(result *SyncResult) string {
	if len(result.ToUpdate) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🔄 Entries that would be updated (IP address): "))
	sb.WriteString("\n")
	for _, hostname := range result.ToUpdate {
		override := result.SyncOverrides[hostname]
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Update.Render("UPDATE "))
		sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
		sb.WriteString(": ")
		sb.WriteString(ui.Styles.IP.Render(override.Server))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ui *SyncUI) RenderUpdateDescEntries(result *SyncResult, entryDescription string) string {
	if len(result.ToUpdateDesc) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 📝 Entries that would be updated (description): "))
	sb.WriteString("\n")
	for _, hostname := range result.ToUpdateDesc {
		override := result.SyncOverrides[hostname]
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Update.Render("UPDATE "))
		sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
		sb.WriteString(": ")
		sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", override.Description)))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", entryDescription)))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ui *SyncUI) RenderRemoveEntries(result *SyncResult) string {
	if len(result.ToRemove) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🗑️ Entries that would be removed: "))
	sb.WriteString("\n")
	for _, hostname := range result.ToRemove {
		override := result.SyncOverrides[hostname]
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Remove.Render("REMOVE "))
		sb.WriteString(ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(override.Server))
		sb.WriteString("\n")
	}
	return sb.String()
}

// cloudflareSyncCmd represents the cloudflare-sync command
var cloudflareSyncCmd = &cobra.Command{
	Use:   "cloudflare-sync",
	Short: "Synchronize DNS entries with Cloudflare tunnel",
	Long: `Synchronize DNS entries in Unbound with hostnames from a Cloudflare tunnel.

This command queries the Cloudflare API for a specified tunnel's configuration,
extracts all hostnames from the ingress routes, and ensures that corresponding
DNS entries exist in Unbound. It will add missing entries, update changed ones
with the correct description, and remove entries that were previously created
by this command but are no longer present in the Cloudflare tunnel.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			// Create sync UI for error message
			syncUI := NewSyncUI()
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf(
						"error loading configuration: %v\nPlease run 'config' command to set up API access",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Create unbound client
		unboundClient := api.NewClient(cfg)

		// Create UI components
		syncUI := NewSyncUI()
		cfUI := NewCloudflareUI()

		// Print header
		fmt.Print(cfUI.RenderHeader())

		// Check for environment variables if flags are not set
		if cfAPIToken == "" {
			cfAPIToken = os.Getenv(EnvCfAPIToken)
			if cfAPIToken != "" && verbose {
				fmt.Println(cfUI.RenderInfo(fmt.Sprintf("Using API token from %s environment variable", EnvCfAPIToken)))
			}
		}

		if cfAccountID == "" {
			cfAccountID = os.Getenv(EnvCfAccountID)
			if cfAccountID != "" && verbose {
				fmt.Println(cfUI.RenderInfo(fmt.Sprintf("Using account ID from %s environment variable", EnvCfAccountID)))
			}
		}

		if cfTunnelID == "" {
			cfTunnelID = os.Getenv(EnvCfTunnelID)
			if cfTunnelID != "" && verbose {
				fmt.Println(cfUI.RenderInfo(fmt.Sprintf("Using tunnel ID from %s environment variable", EnvCfTunnelID)))
			}
		}

		// Verify we have the necessary credentials
		if cfAPIToken == "" {
			fmt.Println(cfUI.RenderError(fmt.Errorf("Cloudflare API token is required")))
			fmt.Println(cfUI.RenderInfo(fmt.Sprintf("You can set it with the --api-token flag or %s environment variable", EnvCfAPIToken)))
			os.Exit(1)
		}

		if cfAccountID == "" {
			fmt.Println(cfUI.RenderError(fmt.Errorf("Cloudflare account ID is required")))
			fmt.Println(cfUI.RenderInfo(fmt.Sprintf("You can set it with the --account-id flag or %s environment variable", EnvCfAccountID)))
			os.Exit(1)
		}

		// Setup cloudflare client
		cfConfig := api.CloudflareConfig{
			APIToken:  cfAPIToken,
			AccountID: cfAccountID,
			TunnelID:  cfTunnelID,
		}

		// Create cloudflare client
		cfClient, err := api.NewCloudflareClient(cfConfig)
		if err != nil {
			logging.Error("Error creating Cloudflare client", "error", err)
			fmt.Println(
				cfUI.RenderError(
					fmt.Errorf("error creating Cloudflare client: %v", err),
				),
			)
			os.Exit(1)
		}

		// If no tunnel ID provided, list available tunnels and exit
		if cfTunnelID == "" {
			fmt.Println(cfUI.RenderInfo("No tunnel ID provided, listing available tunnels..."))
			tunnels, err := cfClient.ListTunnels()
			if err != nil {
				logging.Error("Error listing Cloudflare tunnels", "error", err)
				fmt.Println(
					cfUI.RenderError(
						fmt.Errorf("error listing Cloudflare tunnels: %v", err),
					),
				)
				os.Exit(1)
			}

			if len(tunnels) == 0 {
				fmt.Println(cfUI.RenderWarning("No Cloudflare tunnels found for this account"))
				return
			}

			// Display available tunnels
			fmt.Print(cfUI.RenderTunnelsList(tunnels))
			fmt.Print(cfUI.RenderNoTunnelID())
			return
		}

		// Fetch and process data
		fmt.Println(cfUI.RenderFetchingMessage())

		// Get tunnel details if available
		var tunnelName string
		if verbose {
			tunnels, err := cfClient.ListTunnels()
			if err == nil {
				for _, tunnel := range tunnels {
					if tunnel.ID == cfTunnelID {
						tunnelName = tunnel.Name
						break
					}
				}
				if tunnelName != "" {
					fmt.Println(cfUI.RenderTunnelInfo(cfTunnelID, tunnelName))
				}
			}
		}

		// Perform the sync operation
		options := CloudflareSyncOptions{
			DryRun:             cfDryRun,
			TunnelID:           cfTunnelID,
			EntryDescription:   cfEntryDescription,
			LegacyDescriptions: cfLegacyDescriptions,
			Verbose:            verbose,
		}

		// Perform the sync operation
		result, err := SyncCloudflareWithUnbound(unboundClient, cfClient, options)
		if err != nil {
			logging.Error("Error during sync operation", "error", err)
			fmt.Println(
				cfUI.RenderError(
					fmt.Errorf("error during sync operation: %v", err),
				),
			)
			os.Exit(1)
		}

		// Print source section header
		fmt.Println(cfUI.RenderSourceSection())

		if len(result.HostnameMap) == 0 {
			fmt.Println(cfUI.RenderWarning("No hostnames found in Cloudflare tunnel config"))
			return
		}

		// Display hostname count
		fmt.Print(syncUI.RenderHostnameCount(len(result.HostnameMap)))

		// Display hostnames if verbose
		if verbose {
			fmt.Println()

			// Convert map keys to slice for rendering
			hostnames := make([]string, 0, len(result.HostnameMap))
			for hostname := range result.HostnameMap {
				hostnames = append(hostnames, hostname)
			}

			fmt.Print(syncUI.RenderHostnameList(hostnames))
		}

		// Print summary of changes
		fmt.Print(syncUI.RenderSummary(result))

		// If dry run, just print what would happen and exit
		if cfDryRun {
			fmt.Print(syncUI.RenderDryRunOutput(result, cfEntryDescription))
			return
		}

		// Display changes as they are applied
		fmt.Print(syncUI.RenderChanges(result, cfEntryDescription))
	},
}

func init() {
	rootCmd.AddCommand(cloudflareSyncCmd)

	// Add flags
	cloudflareSyncCmd.Flags().
		BoolVar(&cfDryRun, "dry-run", false, "Show what would be done without making any changes")
	cloudflareSyncCmd.Flags().
		StringVar(&cfTunnelID, "tunnel-id", "", "ID of the Cloudflare tunnel")
	cloudflareSyncCmd.Flags().
		StringVar(&cfAPIToken, "api-token", "", "Cloudflare API token (or set CLOUDFLARE_API_TOKEN)")
	cloudflareSyncCmd.Flags().
		StringVar(&cfAccountID, "account-id", "", "Cloudflare account ID (or set CLOUDFLARE_ACCOUNT_ID)")
	cloudflareSyncCmd.Flags().
		StringVar(&cfEntryDescription, "description", "Entry created by unboundCLI cloudflare-sync",
			"Description to use for created entries")
	cloudflareSyncCmd.Flags().
		StringSliceVar(&cfLegacyDescriptions, "legacy-desc", []string{"Route via Cloudflare"},
			"Legacy descriptions to consider as created by sync")
	cloudflareSyncCmd.Flags().
		BoolVar(&cfForce, "force", false, "Force update even when entries already exist")

	// Add command aliases
	cloudflareSyncCmd.Aliases = []string{"cf-sync"}
}
