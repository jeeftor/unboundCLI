package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/tui"
)

// SyncUI handles the UI rendering for the sync operation
type SyncUI struct {
	Styles tui.StyleConfig
}

// NewSyncUI creates a new SyncUI with default styles
func NewSyncUI() *SyncUI {
	return &SyncUI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the sync operation
func (ui *SyncUI) RenderHeader() string {
	return ui.Styles.Header.Render("✨ Caddy Sync Wizard ✨") + "\n"
}

// RenderSummary renders a summary of the sync operation
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

// RenderDryRun renders the dry run banner
func (ui *SyncUI) RenderDryRun() string {
	return "\n" + ui.Styles.DryRun.Render(" 🧪 DRY RUN - No changes will be made 🧪 ")
}

// RenderAddEntries renders the entries to be added
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

// RenderUpdateEntries renders the entries to be updated
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
		sb.WriteString(
			ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
		)
		sb.WriteString(": ")
		sb.WriteString(ui.Styles.IP.Render(override.Server))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderUpdateDescEntries renders the entries to have their descriptions updated
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
		sb.WriteString(
			ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
		)
		sb.WriteString(": ")
		sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", override.Description)))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", entryDescription)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderRemoveEntries renders the entries to be removed
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
		sb.WriteString(
			ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
		)
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(override.Server))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderDryRunOutput renders the complete dry run output
func (ui *SyncUI) RenderDryRunOutput(result *SyncResult, entryDescription string) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderDryRun())
	sb.WriteString(ui.RenderAddEntries(result))
	sb.WriteString(ui.RenderUpdateEntries(result))
	sb.WriteString(ui.RenderUpdateDescEntries(result, entryDescription))
	sb.WriteString(ui.RenderRemoveEntries(result))

	return sb.String()
}

// RenderChanges renders the changes as they are applied
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
		sb.WriteString(ui.Styles.Section.Render(" 🔄 Updating IP addresses: "))
		sb.WriteString("\n")

		for _, hostname := range result.ToUpdate {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Update.Render("UPDATE "))
			sb.WriteString(
				ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
			)
			sb.WriteString(": ")
			sb.WriteString(ui.Styles.IP.Render(override.Server))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[hostname]))
			sb.WriteString("\n")
		}
	}

	// Update existing entries (description only)
	if len(result.ToUpdateDesc) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 📝 Updating descriptions: "))
		sb.WriteString("\n")

		for _, hostname := range result.ToUpdateDesc {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Update.Render("UPDATE "))
			sb.WriteString(
				ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
			)
			sb.WriteString(": ")
			sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", override.Description)))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.Description.Render(fmt.Sprintf("'%s'", entryDescription)))
			sb.WriteString("\n")
		}
	}

	// Remove stale entries
	if len(result.ToRemove) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🗑️ Removing stale entries: "))
		sb.WriteString("\n")

		for _, hostname := range result.ToRemove {
			override := result.SyncOverrides[hostname]
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Remove.Render("REMOVE "))
			sb.WriteString(
				ui.Styles.Hostname.Render(fmt.Sprintf("%s.%s", override.Host, override.Domain)),
			)
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(override.Server))
			sb.WriteString("\n")
		}
	}

	// Final status message
	if result.ChangesApplied {
		sb.WriteString("\n")
		sb.WriteString("Applying changes... 🛠️\n")
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Success.Render("✨ Changes applied successfully! ✨"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Success.Render("🎉 DNS entries are now in sync with Caddy! 🎉"))
	} else {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Info.Render("🎉 No changes were needed - everything is in sync! 🎉"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Info.Render("😎 Your DNS configuration is already perfect! 😎"))
	}

	return sb.String()
}

// RenderHostnameList renders a list of hostnames
func (ui *SyncUI) RenderHostnameList(hostnames []string) string {
	var sb strings.Builder

	for _, hostname := range hostnames {
		sb.WriteString("  • ")
		sb.WriteString(ui.Styles.Hostname.Render(hostname))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderHostnameCount renders the hostname count
func (ui *SyncUI) RenderHostnameCount(count int) string {
	return fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", count)),
		" hostnames found in Caddy config")
}

// RenderError renders an error message
func (ui *SyncUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" u274c Error: %s ", err))
}

// RenderWarning renders a warning message
func (ui *SyncUI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" u26a0ufe0f  %s ", message))
}

// RenderFetchingMessage renders a message indicating that Caddy config is being fetched
func (ui *SyncUI) RenderFetchingMessage(ip string, port int) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Info.Render(" ud83dudcbe Fetching Caddy config from "))
	sb.WriteString(ui.Styles.IP.Render(fmt.Sprintf("%s:%d", ip, port)))
	sb.WriteString(ui.Styles.Info.Render("... "))

	return sb.String()
}

// Unified sync rendering methods

// RenderUnifiedHeader renders the header for unified sync operations
func (ui *SyncUI) RenderUnifiedHeader(syncToUnbound, syncToAdguard bool) string {
	var targets []string
	if syncToUnbound {
		targets = append(targets, "UnboundDNS")
	}
	if syncToAdguard {
		targets = append(targets, "AdguardHome")
	}

	title := fmt.Sprintf("✨ Unified Caddy Sync (%s) ✨", strings.Join(targets, " + "))
	return ui.Styles.Header.Render(title) + "\n"
}

// RenderSyncTargets renders information about sync targets
func (ui *SyncUI) RenderSyncTargets(syncToUnbound, syncToAdguard bool) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🎯 SYNC TARGETS 🎯 "))
	sb.WriteString("\n\n")

	if syncToUnbound {
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Success.Render("✅ UnboundDNS"))
		sb.WriteString(" - Router-level DNS host overrides\n")
	} else {
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Warning.Render("⏭️ UnboundDNS"))
		sb.WriteString(" - Skipped\n")
	}

	if syncToAdguard {
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Success.Render("✅ AdguardHome"))
		sb.WriteString(" - Client-level DNS rewrites\n")
	} else {
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Warning.Render("⏭️ AdguardHome"))
		sb.WriteString(" - Skipped\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// RenderUnifiedSummary renders a summary of changes for both systems
func (ui *SyncUI) RenderUnifiedSummary(result *UnifiedSyncResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 📊 UNIFIED SYNC SUMMARY 📊 "))
	sb.WriteString("\n\n")

	// UnboundDNS summary
	if result.SyncedToUnbound && result.UnboundResult != nil {
		sb.WriteString(ui.Styles.Info.Render("🔧 UnboundDNS:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.UnboundResult.ToAdd))),
			ui.Styles.Add.Render("✨ Overrides to add")))
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.UnboundResult.ToUpdate))),
			ui.Styles.Update.Render("🔄 Overrides to update IP")))
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.UnboundResult.ToUpdateDesc))),
			ui.Styles.Update.Render("📝 Overrides to update description")))
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.UnboundResult.ToRemove))),
			ui.Styles.Remove.Render("🗑️ Overrides to remove")))
	} else if result.UnboundError != nil {
		sb.WriteString(ui.Styles.Error.Render("❌ UnboundDNS: FAILED"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Error: %v\n", result.UnboundError))
	}

	// AdguardHome summary
	if result.SyncedToAdguard && result.AdguardResult != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Info.Render("🛡️ AdguardHome:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.AdguardResult.ToAdd))),
			ui.Styles.Add.Render("✨ Rewrites to add")))
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.AdguardResult.ToUpdate))),
			ui.Styles.Update.Render("🔄 Rewrites to update")))
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.AdguardResult.ToRemove))),
			ui.Styles.Remove.Render("🗑️ Rewrites to remove")))
	} else if result.AdguardError != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Error.Render("❌ AdguardHome: FAILED"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Error: %v\n", result.AdguardError))
	}

	return sb.String()
}

// RenderUnifiedDryRunOutput renders the complete unified dry run output
func (ui *SyncUI) RenderUnifiedDryRunOutput(result *UnifiedSyncResult, entryDescription string) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderDryRun())

	// UnboundDNS dry run output
	if result.SyncedToUnbound && result.UnboundResult != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🔧 UNBOUND DNS CHANGES 🔧 "))
		sb.WriteString(ui.RenderAddEntries(result.UnboundResult))
		sb.WriteString(ui.RenderUpdateEntries(result.UnboundResult))
		sb.WriteString(ui.RenderUpdateDescEntries(result.UnboundResult, entryDescription))
		sb.WriteString(ui.RenderRemoveEntries(result.UnboundResult))
	}

	// AdguardHome dry run output
	if result.SyncedToAdguard && result.AdguardResult != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🛡️ ADGUARD HOME CHANGES 🛡️ "))
		sb.WriteString(ui.RenderAdguardAddEntries(result.AdguardResult))
		sb.WriteString(ui.RenderAdguardUpdateEntries(result.AdguardResult))
		sb.WriteString(ui.RenderAdguardRemoveEntries(result.AdguardResult))
	}

	return sb.String()
}

// RenderUnifiedChanges renders the unified changes as they are applied
func (ui *SyncUI) RenderUnifiedChanges(result *UnifiedSyncResult, entryDescription string) string {
	var sb strings.Builder

	// UnboundDNS changes
	if result.SyncedToUnbound && result.UnboundResult != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🔧 APPLYING UNBOUND DNS CHANGES 🔧 "))

		hasChanges := len(result.UnboundResult.ToAdd) > 0 ||
			len(result.UnboundResult.ToUpdate) > 0 ||
			len(result.UnboundResult.ToUpdateDesc) > 0 ||
			len(result.UnboundResult.ToRemove) > 0

		if hasChanges {
			sb.WriteString(ui.RenderChanges(result.UnboundResult, entryDescription))
		} else {
			sb.WriteString("\n")
			sb.WriteString(ui.Styles.Info.Render("🎉 No changes needed - UnboundDNS is in sync! 🎉"))
		}
	}

	// AdguardHome changes
	if result.SyncedToAdguard && result.AdguardResult != nil {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🛡️ APPLYING ADGUARD HOME CHANGES 🛡️ "))

		hasChanges := len(result.AdguardResult.ToAdd) > 0 ||
			len(result.AdguardResult.ToUpdate) > 0 ||
			len(result.AdguardResult.ToRemove) > 0

		if hasChanges {
			sb.WriteString(ui.RenderAdguardChanges(result.AdguardResult, entryDescription))
		} else {
			sb.WriteString("\n")
			sb.WriteString(ui.Styles.Info.Render("🎉 No changes needed - AdguardHome is in sync! 🎉"))
		}
	}

	// Final unified status
	bothSuccessful := (result.UnboundResult == nil || result.UnboundResult.ChangesApplied ||
		(len(result.UnboundResult.ToAdd) == 0 && len(result.UnboundResult.ToUpdate) == 0 &&
			len(result.UnboundResult.ToUpdateDesc) == 0 && len(result.UnboundResult.ToRemove) == 0)) &&
		(result.AdguardResult == nil || result.AdguardResult.ChangesApplied ||
			(len(result.AdguardResult.ToAdd) == 0 && len(result.AdguardResult.ToUpdate) == 0 &&
				len(result.AdguardResult.ToRemove) == 0))

	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🎊 UNIFIED SYNC COMPLETE 🎊 "))
	sb.WriteString("\n\n")

	if bothSuccessful {
		sb.WriteString(ui.Styles.Success.Render("✨ All systems synchronized with Caddy! ✨"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Success.Render("🌟 Split-horizon DNS is now fully configured! 🌟"))
		sb.WriteString("\n")
		sb.WriteString("  - UnboundDNS: Router-level *.vookie.net → Caddy\n")
		sb.WriteString("  - AdguardHome: Client-level *.vookie.net → Caddy\n")
	} else {
		sb.WriteString(ui.Styles.Info.Render("🎯 Sync completed with mixed results"))
		sb.WriteString("\n")
		if result.UnboundError != nil {
			sb.WriteString(fmt.Sprintf("  ❌ UnboundDNS: %v\n", result.UnboundError))
		}
		if result.AdguardError != nil {
			sb.WriteString(fmt.Sprintf("  ❌ AdguardHome: %v\n", result.AdguardError))
		}
	}

	return sb.String()
}

// AdguardHome-specific rendering methods

// RenderAdguardSummary renders a summary of the AdguardHome sync operation
func (ui *SyncUI) RenderAdguardSummary(result *AdguardSyncResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 📊 ADGUARD SYNC SUMMARY 📊 "))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToAdd))),
		ui.Styles.Add.Render(" ✨ Rewrites to add")))

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToUpdate))),
		ui.Styles.Update.Render(" 🔄 Rewrites to update")))

	sb.WriteString(fmt.Sprintf("%s %s\n",
		ui.Styles.Count.Render(fmt.Sprintf("%d", len(result.ToRemove))),
		ui.Styles.Remove.Render(" 🗑️ Rewrites to remove")))

	return sb.String()
}

// RenderAdguardAddEntries renders the AdguardHome rewrites to be added
func (ui *SyncUI) RenderAdguardAddEntries(result *AdguardSyncResult) string {
	if len(result.ToAdd) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" ✨ Rewrites that would be added: "))
	sb.WriteString("\n")

	for _, domain := range result.ToAdd {
		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Add.Render("ADD "))
		sb.WriteString(ui.Styles.Hostname.Render(domain))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[domain]))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderAdguardUpdateEntries renders the AdguardHome rewrites to be updated
func (ui *SyncUI) RenderAdguardUpdateEntries(result *AdguardSyncResult) string {
	if len(result.ToUpdate) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🔄 Rewrites that would be updated: "))
	sb.WriteString("\n")

	for _, domain := range result.ToUpdate {
		// Find the existing rewrite for this domain
		var oldAnswer string
		for _, rewrite := range result.SyncRewrites {
			if rewrite.Domain == domain {
				oldAnswer = rewrite.Answer
				break
			}
		}

		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Update.Render("UPDATE "))
		sb.WriteString(ui.Styles.Hostname.Render(domain))
		sb.WriteString(": ")
		sb.WriteString(ui.Styles.IP.Render(oldAnswer))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[domain]))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderAdguardRemoveEntries renders the AdguardHome rewrites to be removed
func (ui *SyncUI) RenderAdguardRemoveEntries(result *AdguardSyncResult) string {
	if len(result.ToRemove) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Section.Render(" 🗑️ Rewrites that would be removed: "))
	sb.WriteString("\n")

	for _, domain := range result.ToRemove {
		// Find the existing rewrite for this domain
		var answer string
		for _, rewrite := range result.SyncRewrites {
			if rewrite.Domain == domain {
				answer = rewrite.Answer
				break
			}
		}

		sb.WriteString("  ")
		sb.WriteString(ui.Styles.Remove.Render("REMOVE "))
		sb.WriteString(ui.Styles.Hostname.Render(domain))
		sb.WriteString(" → ")
		sb.WriteString(ui.Styles.IP.Render(answer))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderAdguardDryRunOutput renders the complete AdguardHome dry run output
func (ui *SyncUI) RenderAdguardDryRunOutput(result *AdguardSyncResult, entryDescription string) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderDryRun())
	sb.WriteString(ui.RenderAdguardAddEntries(result))
	sb.WriteString(ui.RenderAdguardUpdateEntries(result))
	sb.WriteString(ui.RenderAdguardRemoveEntries(result))

	return sb.String()
}

// RenderAdguardChanges renders the AdguardHome changes as they are applied
func (ui *SyncUI) RenderAdguardChanges(result *AdguardSyncResult, entryDescription string) string {
	var sb strings.Builder

	// Add new rewrites
	if len(result.ToAdd) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" ✨ Adding new rewrites: "))
		sb.WriteString("\n")

		for _, domain := range result.ToAdd {
			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Add.Render("ADD "))
			sb.WriteString(ui.Styles.Hostname.Render(domain))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[domain]))
			sb.WriteString("\n")
		}
	}

	// Update existing rewrites
	if len(result.ToUpdate) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🔄 Updating rewrites: "))
		sb.WriteString("\n")

		for _, domain := range result.ToUpdate {
			// Find the existing rewrite for this domain
			var oldAnswer string
			for _, rewrite := range result.SyncRewrites {
				if rewrite.Domain == domain {
					oldAnswer = rewrite.Answer
					break
				}
			}

			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Update.Render("UPDATE "))
			sb.WriteString(ui.Styles.Hostname.Render(domain))
			sb.WriteString(": ")
			sb.WriteString(ui.Styles.IP.Render(oldAnswer))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(result.HostnameMap[domain]))
			sb.WriteString("\n")
		}
	}

	// Remove stale rewrites
	if len(result.ToRemove) > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Section.Render(" 🗑️ Removing stale rewrites: "))
		sb.WriteString("\n")

		for _, domain := range result.ToRemove {
			// Find the existing rewrite for this domain
			var answer string
			for _, rewrite := range result.SyncRewrites {
				if rewrite.Domain == domain {
					answer = rewrite.Answer
					break
				}
			}

			sb.WriteString("  ")
			sb.WriteString(ui.Styles.Remove.Render("REMOVE "))
			sb.WriteString(ui.Styles.Hostname.Render(domain))
			sb.WriteString(" → ")
			sb.WriteString(ui.Styles.IP.Render(answer))
			sb.WriteString("\n")
		}
	}

	// Final status message
	if result.ChangesApplied {
		sb.WriteString("\n")
		sb.WriteString("Applying changes... 🛠️\n")
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Success.Render("✨ AdguardHome rewrites updated successfully! ✨"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Success.Render("🎉 DNS rewrites are now in sync with Caddy! 🎉"))
	} else {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Info.Render("🎉 No changes were needed - rewrites are in sync! 🎉"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Info.Render("😎 Your AdguardHome DNS rewrites are already perfect! 😎"))
	}

	return sb.String()
}

// RenderCloudflareHeader renders the header for Cloudflare sync operation
func (ui *SyncUI) RenderCloudflareHeader(syncDirect, syncCaddy bool) string {
	var sb strings.Builder

	sb.WriteString("🚀 Caddy → Cloudflare Dual-Mode DNS Sync 🚀\n")

	if syncDirect && syncCaddy {
		sb.WriteString("Syncing hostnames to both direct and Caddy proxy routing\n")
	} else if syncDirect {
		sb.WriteString("Syncing hostnames for direct service access only\n")
	} else if syncCaddy {
		sb.WriteString("Syncing hostnames for Caddy proxy access only\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// RenderCloudflareSyncTargets renders the sync targets for Cloudflare operation
func (ui *SyncUI) RenderCloudflareSyncTargets(syncDirect, syncCaddy bool, directSub, caddySub string) string {
	var sb strings.Builder

	sb.WriteString("📋 Sync Targets:\n")

	if syncDirect {
		sb.WriteString(fmt.Sprintf("  🎯 Direct Access: service.%s.vookie.net → Service IP\n", directSub))
	}
	if syncCaddy {
		sb.WriteString(fmt.Sprintf("  🔄 Caddy Proxy: service.%s.vookie.net → Caddy IP\n", caddySub))
	}

	sb.WriteString("\n")
	return sb.String()
}

// RenderCloudflareSummary renders the summary of changes for Cloudflare sync
func (ui *SyncUI) RenderCloudflareSummary(result *CaddyCloudflareSyncResult) string {
	var sb strings.Builder

	sb.WriteString("\n📊 SUMMARY OF CHANGES TO UnboundDNS\n")
	sb.WriteString(fmt.Sprintf("✨ %d DNS overrides to add (new Cloudflare routes)\n", len(result.ToAdd)))
	sb.WriteString(fmt.Sprintf("🔄 %d DNS overrides to update (IP address changed)\n", len(result.ToUpdate)))
	sb.WriteString(fmt.Sprintf("🗑️  %d DNS overrides to remove (no longer in Caddy)\n", len(result.ToRemove)))

	directCount := len(result.DirectEntries)
	caddyCount := len(result.CaddyEntries)
	sb.WriteString(fmt.Sprintf("📋 %d direct access entries, %d Caddy proxy entries configured\n", directCount, caddyCount))

	return sb.String()
}

// RenderCloudflareDryRunOutput renders what would happen in dry run mode
func (ui *SyncUI) RenderCloudflareDryRunOutput(result *CaddyCloudflareSyncResult, description string) string {
	var sb strings.Builder

	sb.WriteString("\n🧪 DRY RUN - No changes will be made\n")

	// Display additions
	if len(result.ToAdd) > 0 {
		sb.WriteString(fmt.Sprintf("\n✨ Would add %d DNS overrides to UnboundDNS:\n", len(result.ToAdd)))
		for _, entry := range result.ToAdd {
			modeIcon := "🎯"
			if entry.Mode == "caddy" {
				modeIcon = "🔄"
			}
			sb.WriteString(fmt.Sprintf("  %s %s.%s → %s (%s mode)\n",
				modeIcon, entry.Hostname, entry.Domain, entry.IP, entry.Mode))
		}
	}

	// Display updates
	if len(result.ToUpdate) > 0 {
		sb.WriteString(fmt.Sprintf("\n🔄 Would update %d DNS overrides in UnboundDNS:\n", len(result.ToUpdate)))
		for _, entry := range result.ToUpdate {
			modeIcon := "🎯"
			if entry.Mode == "caddy" {
				modeIcon = "🔄"
			}
			sb.WriteString(fmt.Sprintf("  %s %s.%s → %s (%s mode)\n",
				modeIcon, entry.Hostname, entry.Domain, entry.IP, entry.Mode))
		}
	}

	// Display removals
	if len(result.ToRemove) > 0 {
		sb.WriteString(fmt.Sprintf("\n🗑️  Would remove %d DNS overrides from UnboundDNS:\n", len(result.ToRemove)))
		for _, override := range result.ToRemove {
			sb.WriteString(fmt.Sprintf("  • %s.%s → %s (no longer in Caddy)\n",
				override.Host, override.Domain, override.Server))
		}
	}

	return sb.String()
}

// RenderCloudflareChanges renders the changes as they are applied
func (ui *SyncUI) RenderCloudflareChanges(result *CaddyCloudflareSyncResult, description string) string {
	var sb strings.Builder

	if len(result.ToAdd)+len(result.ToUpdate)+len(result.ToRemove) == 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Info.Render("🎉 No changes needed - DNS overrides are already in sync! 🎉"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Info.Render("😎 Your Cloudflare tunnel DNS routes are already perfect! 😎"))
		return sb.String()
	}

	if result.ChangesApplied {
		sb.WriteString("\n")
		sb.WriteString("Applying changes... 🛠️\n")
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Success.Render("✨ Cloudflare tunnel DNS routes updated successfully! ✨"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Success.Render("🎉 DNS overrides are now in sync for dual-mode access! 🎉"))
	} else {
		sb.WriteString("\n")
		sb.WriteString(ui.Styles.Info.Render("🎉 No changes were needed - routes are in sync! 🎉"))
		sb.WriteString("\n\n")
		sb.WriteString(ui.Styles.Info.Render("😎 Your Cloudflare tunnel DNS routes are already perfect! 😎"))
	}

	return sb.String()
}
