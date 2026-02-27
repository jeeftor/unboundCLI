package commands

import (
	"fmt"
	"reflect"

	"github.com/jeeftor/caddy-dns-sync/internal/tables"
)

// ListDataSource defines interface for data fetching
type ListDataSource interface {
	// Initialize loads config and creates client
	Initialize() error

	// FetchData fetches data from the target system
	FetchData() (interface{}, error)

	// FormatAsTable formats data as table rows
	FormatAsTable() tables.TableConfig

	// FormatAsJSON formats data as JSON
	FormatAsJSON() ([]byte, error)

	// EmptyMessage returns message when no data is found
	EmptyMessage() string
}

// ListCommandRunner executes list operations
type ListCommandRunner struct {
	source     ListDataSource
	jsonOutput bool
	quietMode  bool
	renderer   *tables.TableRenderer
}

// NewListCommandRunner creates a new list command runner
func NewListCommandRunner(source ListDataSource) *ListCommandRunner {
	return &ListCommandRunner{
		source:   source,
		renderer: tables.NewTableRenderer(),
	}
}

// SetJSONOutput sets whether to output JSON
func (r *ListCommandRunner) SetJSONOutput(enabled bool) {
	r.jsonOutput = enabled
}

// SetQuietMode sets whether to suppress informational output
func (r *ListCommandRunner) SetQuietMode(enabled bool) {
	r.quietMode = enabled
}

// Run executes the list command
func (r *ListCommandRunner) Run() error {
	// Initialize (load config, create client)
	if err := r.source.Initialize(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Fetch data
	if !r.quietMode {
		fmt.Println("Fetching data...")
	}

	data, err := r.source.FetchData()
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}

	// Handle empty result
	if isEmpty(data) {
		if !r.quietMode {
			fmt.Println(r.source.EmptyMessage())
		}
		return nil
	}

	// Output as JSON or table
	if r.jsonOutput {
		jsonData, err := r.source.FormatAsJSON()
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		tableCfg := r.source.FormatAsTable()
		fmt.Print(r.renderer.Render(tableCfg))
	}

	return nil
}

// isEmpty checks if data is empty using reflection
func isEmpty(data interface{}) bool {
	if data == nil {
		return true
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem().Interface())
	default:
		return false
	}
}
