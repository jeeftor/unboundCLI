package widgets

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

func TestTableWidgetSelectionFollowsEntryAfterSorting(t *testing.T) {
	w := NewTableWidget()
	w.SetSize(120, 20)
	w.SetEntries([]*models.Entry{
		{Hostname: "alpha.example.com", CaddyUpstream: "10.0.0.2:80"},
		{Hostname: "beta.example.com", CaddyUpstream: "10.0.0.1:80"},
	})

	w.cursor = 0
	w.toggleSelection()
	w.CycleSort()

	selected := w.GetSelectedEntries()
	if len(selected) != 1 {
		t.Fatalf("expected one selected entry, got %d", len(selected))
	}
	if selected[0].Hostname != "alpha.example.com" {
		t.Fatalf("selection moved after sorting: got %q", selected[0].Hostname)
	}
}
