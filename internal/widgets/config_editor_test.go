package widgets

import (
	"testing"
)

func makeTestEditor() *ConfigEditorWidget {
	editor := NewConfigEditor()
	editor.SetSize(80, 40)
	editor.SetSections([]ConfigSection{
		{
			Title: "Credentials",
			Fields: []ConfigField{
				{Key: "username", Label: "Username", Value: "admin"},
				{Key: "password", Label: "Password", Value: "s3cr3t", IsPassword: true},
				{Key: "api_key", Label: "API Key", Value: "mykey123", IsPassword: true},
			},
		},
		{
			Title: "Server",
			Fields: []ConfigField{
				{Key: "base_url", Label: "Base URL", Value: "https://192.168.1.1"},
				{Key: "insecure", Label: "Skip SSL", Value: "false"},
			},
		},
	})
	return editor
}

func TestPasswordFieldsHiddenByDefault(t *testing.T) {
	editor := makeTestEditor()

	if editor.ShowingPasswords() {
		t.Error("passwords should be hidden by default")
	}
}

func TestTogglePasswordVisibility_ShowThenHide(t *testing.T) {
	editor := makeTestEditor()

	editor.TogglePasswordVisibility()
	if !editor.ShowingPasswords() {
		t.Error("ShowingPasswords should be true after first toggle")
	}

	editor.TogglePasswordVisibility()
	if editor.ShowingPasswords() {
		t.Error("ShowingPasswords should be false after second toggle")
	}
}

func TestTogglePasswordVisibility_PreservesValues(t *testing.T) {
	editor := makeTestEditor()

	// Toggle show
	editor.TogglePasswordVisibility()
	vals := editor.GetAllValues()
	if vals["password"] != "s3cr3t" {
		t.Errorf("password lost after show toggle: got %q, want %q", vals["password"], "s3cr3t")
	}
	if vals["api_key"] != "mykey123" {
		t.Errorf("api_key lost after show toggle: got %q, want %q", vals["api_key"], "mykey123")
	}
	if vals["username"] != "admin" {
		t.Errorf("username changed after show toggle: got %q", vals["username"])
	}

	// Toggle hide
	editor.TogglePasswordVisibility()
	vals = editor.GetAllValues()
	if vals["password"] != "s3cr3t" {
		t.Errorf("password lost after hide toggle: got %q, want %q", vals["password"], "s3cr3t")
	}
	if vals["api_key"] != "mykey123" {
		t.Errorf("api_key lost after hide toggle: got %q, want %q", vals["api_key"], "mykey123")
	}
}

func TestGetAllValues_ReturnsAllFields(t *testing.T) {
	editor := makeTestEditor()
	vals := editor.GetAllValues()

	expected := map[string]string{
		"username": "admin",
		"password": "s3cr3t",
		"api_key":  "mykey123",
		"base_url": "https://192.168.1.1",
		"insecure": "false",
	}
	for key, want := range expected {
		if got := vals[key]; got != want {
			t.Errorf("GetAllValues()[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestSetValue_UpdatesField(t *testing.T) {
	editor := makeTestEditor()
	editor.SetValue("base_url", "https://10.0.0.1")

	vals := editor.GetAllValues()
	if vals["base_url"] != "https://10.0.0.1" {
		t.Errorf("SetValue did not update field: got %q", vals["base_url"])
	}
	// Other fields unchanged
	if vals["username"] != "admin" {
		t.Errorf("SetValue changed unrelated field: got %q", vals["username"])
	}
}

func TestGetValue_ReturnsCorrectValue(t *testing.T) {
	editor := makeTestEditor()

	if got := editor.GetValue("username"); got != "admin" {
		t.Errorf("GetValue(username) = %q, want %q", got, "admin")
	}
	if got := editor.GetValue("nonexistent"); got != "" {
		t.Errorf("GetValue(nonexistent) = %q, want empty string", got)
	}
}

func TestMultipleTogglesCycleCorrectly(t *testing.T) {
	editor := makeTestEditor()

	for i := 0; i < 6; i++ {
		editor.TogglePasswordVisibility()
		wantShowing := (i+1)%2 == 1
		if editor.ShowingPasswords() != wantShowing {
			t.Errorf("toggle %d: ShowingPasswords() = %v, want %v", i+1, editor.ShowingPasswords(), wantShowing)
		}
	}
}
