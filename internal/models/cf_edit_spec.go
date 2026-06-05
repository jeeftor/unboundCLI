package models

// CFEditSpec carries the user's desired CF ingress rule settings from the edit widget.
type CFEditSpec struct {
	Hostname       string
	Service        string
	HTTPHostHeader string
	NoTLSVerify    bool
	Http2Origin    bool
}
