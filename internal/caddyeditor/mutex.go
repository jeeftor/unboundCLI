package caddyeditor

import "sync"

// editorMu serializes all Caddyfile mutations and deploy operations.
// This prevents concurrent edits from corrupting the file or git state.
var editorMu sync.Mutex
