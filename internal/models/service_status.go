package models

// ServiceStatus represents the configuration status of a DNS service (Unbound or AdGuard)
type ServiceStatus struct {
	Configured bool   // Is this service configured for this hostname?
	IP         string // What IP is configured (if any)
	InSync     bool   // Does the configured IP match the expected IP?
}

// NewServiceStatus creates a new ServiceStatus
func NewServiceStatus(configured bool, ip string, inSync bool) ServiceStatus {
	return ServiceStatus{
		Configured: configured,
		IP:         ip,
		InSync:     inSync,
	}
}

// NotConfigured returns a ServiceStatus for an unconfigured service
func NotConfigured() ServiceStatus {
	return ServiceStatus{
		Configured: false,
		IP:         "",
		InSync:     false,
	}
}

// Synced returns a ServiceStatus for a properly synced service
func Synced(ip string) ServiceStatus {
	return ServiceStatus{
		Configured: true,
		IP:         ip,
		InSync:     true,
	}
}

// NotInSync returns a ServiceStatus for an out-of-sync service
func NotInSync(ip string) ServiceStatus {
	return ServiceStatus{
		Configured: true,
		IP:         ip,
		InSync:     false,
	}
}

// DHCPStatus represents DHCP lease information
type DHCPStatus struct {
	Configured bool   // Is there a DHCP lease for this device?
	Type       string // "static" or "dynamic"
	IP         string // Leased IP address
	MAC        string // MAC address
	Hostname   string // Hostname from DHCP
	InSync     bool   // Does DHCP IP match expected IP?
}

// NewDHCPStatus creates a new DHCPStatus
func NewDHCPStatus(configured bool, leaseType, ip, mac, hostname string, inSync bool) DHCPStatus {
	return DHCPStatus{
		Configured: configured,
		Type:       leaseType,
		IP:         ip,
		MAC:        mac,
		Hostname:   hostname,
		InSync:     inSync,
	}
}

// NoDHCP returns a DHCPStatus for when no DHCP lease exists
func NoDHCP() DHCPStatus {
	return DHCPStatus{
		Configured: false,
		Type:       "",
		IP:         "",
		MAC:        "",
		Hostname:   "",
		InSync:     false,
	}
}

// IsStatic returns true if this is a static DHCP reservation
func (d DHCPStatus) IsStatic() bool {
	return d.Configured && d.Type == "static"
}

// IsDynamic returns true if this is a dynamic DHCP lease
func (d DHCPStatus) IsDynamic() bool {
	return d.Configured && d.Type == "dynamic"
}
