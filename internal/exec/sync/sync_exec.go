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
