package status

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

type ServiceName string

const (
	ServiceCaddy      ServiceName = "caddy"
	ServiceUnbound    ServiceName = "unbound"
	ServiceAdguard    ServiceName = "adguard"
	ServiceDHCP       ServiceName = "dhcp"
	ServiceCloudflare ServiceName = "cloudflare"
	ServiceDNS        ServiceName = "dns"
)

type ServiceState string

const (
	ServicePending ServiceState = "pending"
	ServiceLoaded  ServiceState = "loaded"
	ServiceSkipped ServiceState = "skipped"
	ServiceFailed  ServiceState = "failed"
)

type ServiceReport struct {
	Status ServiceState `json:"status"`
	Count  int          `json:"count"`
	Error  string       `json:"error"`
}

type LoadReport struct {
	Services map[ServiceName]ServiceReport `json:"services"`
}

type ProgressEvent struct {
	Service ServiceName  `json:"service"`
	Status  ServiceState `json:"status"`
	Count   int          `json:"count"`
	Error   string       `json:"error"`
}

type Options struct {
	CaddyServerIP string
	Progress      func(ProgressEvent)
}

// DataLoader handles loading data from all API clients and building unified Entry models
type DataLoader struct {
	caddyClient   *api.CaddyClient
	unboundClient *api.Client
	adguardClient *api.AdguardClient
	dnsmasqClient *api.DNSMasqClient
	cfClient      *api.CloudflareClient
	caddyServerIP string
	progress      func(ProgressEvent)
	ctx           context.Context
}

func LoadEntries(ctx context.Context, clients app.ClientSet, options Options) ([]*models.Entry, LoadReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := NewDataLoader(
		clients.Caddy,
		clients.Unbound,
		clients.Adguard,
		clients.DNSMasq,
		options.CaddyServerIP,
	)
	loader.WithCloudflareClient(clients.Cloudflare)
	loader.WithContext(ctx)
	loader.progress = options.Progress
	return loader.LoadDataWithReport()
}

// WithContext sets the cancellation context for load phases that support it.
func (d *DataLoader) WithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	d.ctx = ctx
}

// WithCloudflareClient sets an optional Cloudflare client for loading CF tunnel data.
// If nil, CF data is skipped and the TUI works without it.
func (d *DataLoader) WithCloudflareClient(c *api.CloudflareClient) {
	d.cfClient = c
}

// WithProgress sets a callback that's invoked as each service loads.
// The callback receives a ProgressEvent with the service name, status
// (pending/loaded/failed/skipped), count, and any error message.
func (d *DataLoader) WithProgress(fn func(ProgressEvent)) {
	d.progress = fn
}

// NewDataLoader creates a new data loader
func NewDataLoader(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
	caddyServerIP string,
) *DataLoader {
	return &DataLoader{
		caddyClient:   caddyClient,
		unboundClient: unboundClient,
		adguardClient: adguardClient,
		dnsmasqClient: dnsmasqClient,
		caddyServerIP: caddyServerIP,
		ctx:           context.Background(),
	}
}

// LoadData loads all data from API clients concurrently and builds Entry models.
// Caddy, Unbound, AdGuard, DHCP, and Cloudflare are fetched in parallel.
// DNS resolution for all entries is also parallelized with a worker pool.
func (d *DataLoader) LoadData() ([]*models.Entry, error) {
	entries, _, err := d.LoadDataWithReport()
	return entries, err
}

func (d *DataLoader) LoadDataWithReport() ([]*models.Entry, LoadReport, error) {
	report := newLoadReport()
	for _, service := range loadReportServices() {
		d.emit(ProgressEvent{Service: service, Status: ServicePending})
	}
	if err := d.contextErr(); err != nil {
		report.markUnfinished(ServiceFailed, err.Error())
		d.emitAllReports(report)
		return nil, report, err
	}

	// --- Phase 1: parallel API fetches ---
	data, errs := d.fetchAllData()

	if err := d.contextErr(); err != nil {
		report.markUnfinished(ServiceFailed, err.Error())
		d.emitAllReports(report)
		return nil, report, err
	}

	report.set(ServiceCaddy, serviceReport(data.caddyHostnames, errs.caddy, false))
	report.set(ServiceUnbound, serviceReport(data.unboundOverrides, errs.unbound, d.unboundClient == nil))
	report.set(ServiceAdguard, serviceReport(data.adguardRewrites, errs.adguard, d.adguardClient == nil))
	report.set(ServiceDHCP, serviceReport(data.dhcpLeases, errs.dhcp, d.dnsmasqClient == nil))
	if report.Services[ServiceDHCP].Status == ServiceLoaded {
		dhcpReport := report.Services[ServiceDHCP]
		dhcpReport.Count = data.dhcpLeaseCount
		report.set(ServiceDHCP, dhcpReport)
	}
	report.set(ServiceCloudflare, serviceReport(data.cfDetails, errs.cf, d.cfClient == nil))
	d.emitReports(report, []ServiceName{ServiceCaddy, ServiceUnbound, ServiceAdguard, ServiceDHCP, ServiceCloudflare})

	if errs.caddy != nil {
		report.set(ServiceDNS, ServiceReport{Status: ServiceSkipped, Error: "skipped because Caddy load failed"})
		d.emitServiceReport(ServiceDNS, report.Services[ServiceDNS])
		return nil, report, fmt.Errorf("failed to load Caddy hostnames: %w", errs.caddy)
	}
	if err := d.contextErr(); err != nil {
		report.markUnfinished(ServiceFailed, err.Error())
		d.emitAllReports(report)
		return nil, report, err
	}

	// --- Phase 2: build entry models ---
	logging.Info("Building unified entry models...")
	entries := d.buildEntries(data.caddyHostnames, data.unboundOverrides, data.adguardRewrites, data.dhcpLeases)
	logging.Info("Built entry models", "count", len(entries))

	// --- Phase 3: parallel DNS resolution ---
	logging.Info("Resolving DNS hostnames in parallel...")
	if err := d.contextErr(); err != nil {
		report.markUnfinished(ServiceFailed, err.Error())
		d.emitAllReports(report)
		return nil, report, err
	}
	d.resolveAllDNS(entries)
	report.set(ServiceDNS, ServiceReport{Status: ServiceLoaded, Count: len(entries)})
	d.emitServiceReport(ServiceDNS, report.Services[ServiceDNS])

	// --- Phase 4: enrich with Cloudflare data ---
	d.enrichWithCloudflare(entries, data.cfDetails, data.cfDNSRecords)

	// Recompute overall status now that CF data is merged
	for _, e := range entries {
		e.OverallStatus = models.ComputeSyncStatus(e, d.caddyServerIP)
	}

	d.logLoadSummary(entries)

	return entries, report, nil
}

// fetchedData holds the results of parallel API fetches.
type fetchedData struct {
	caddyHostnames   map[string]models.CaddyRouteInfo
	unboundOverrides map[string]*api.DNSOverride
	adguardRewrites  map[string]*api.Rewrite
	dhcpLeases       map[string]*api.DNSMasqLease
	dhcpLeaseCount   int
	cfDetails        map[string]api.CloudflareIngressEntry
	cfDNSRecords     map[string]string
}

// fetchErrors holds errors from parallel API fetches.
type fetchErrors struct {
	caddy   error
	unbound error
	adguard error
	dhcp    error
	cf      error
}

// fetchAllData runs all API fetches in parallel and returns the collected data and errors.
func (d *DataLoader) fetchAllData() (fetchedData, fetchErrors) {
	var (
		data fetchedData
		errs fetchErrors
		wg   sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.contextErr() != nil {
			return
		}
		logging.Info("Loading Caddy configuration...")
		data.caddyHostnames, errs.caddy = d.loadCaddyHostnames()
		if errs.caddy != nil {
			logging.Error("Failed to load Caddy hostnames", "error", errs.caddy)
		} else {
			logging.Info("Loaded Caddy hostnames", "count", len(data.caddyHostnames))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.contextErr() != nil {
			return
		}
		logging.Info("Loading Unbound DNS overrides...")
		if d.unboundClient == nil {
			data.unboundOverrides = make(map[string]*api.DNSOverride)
			return
		}
		data.unboundOverrides, errs.unbound = d.loadUnboundOverrides()
		if errs.unbound != nil {
			logging.Warn("Failed to load Unbound overrides", "error", errs.unbound)
			data.unboundOverrides = make(map[string]*api.DNSOverride)
		} else {
			logging.Info("Loaded Unbound overrides", "count", len(data.unboundOverrides))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.contextErr() != nil {
			return
		}
		if d.adguardClient == nil {
			data.adguardRewrites = make(map[string]*api.Rewrite)
			return
		}
		logging.Info("Loading AdGuard DNS rewrites...")
		data.adguardRewrites, errs.adguard = d.loadAdguardRewrites()
		if errs.adguard != nil {
			logging.Warn("Failed to load AdGuard rewrites", "error", errs.adguard)
			data.adguardRewrites = make(map[string]*api.Rewrite)
		} else {
			logging.Info("Loaded AdGuard rewrites", "count", len(data.adguardRewrites))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.contextErr() != nil {
			return
		}
		logging.Info("Loading DHCP leases...")
		if d.dnsmasqClient == nil {
			data.dhcpLeases = make(map[string]*api.DNSMasqLease)
			return
		}
		data.dhcpLeases, errs.dhcp = d.loadDHCPLeases()
		data.dhcpLeaseCount = countUniqueDHCPLeases(data.dhcpLeases)
		if errs.dhcp != nil {
			logging.Warn("Failed to load DHCP leases", "error", errs.dhcp)
			data.dhcpLeases = make(map[string]*api.DNSMasqLease)
			data.dhcpLeaseCount = 0
		} else {
			logging.Info("Loaded DHCP leases", "count", data.dhcpLeaseCount)
		}
	}()

	if d.cfClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if d.contextErr() != nil {
				return
			}
			logging.Info("Loading Cloudflare tunnel details...")
			data.cfDetails, errs.cf = d.cfClient.GetAllTunnelsDetails()
			if errs.cf != nil {
				logging.Warn("Failed to load Cloudflare tunnel details", "error", errs.cf)
			} else {
				logging.Info("Loaded Cloudflare tunnel details", "count", len(data.cfDetails))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if d.contextErr() != nil {
				return
			}
			records, err := d.cfClient.ListManagedDNSRecords()
			if err != nil {
				logging.Warn("Failed to load Cloudflare DNS records", "error", err)
			} else {
				data.cfDNSRecords = records
				logging.Info("Loaded Cloudflare DNS records", "count", len(records))
			}
		}()
	}

	wg.Wait()
	return data, errs
}

// enrichWithCloudflare merges Cloudflare tunnel and DNS data into the entries.
func (d *DataLoader) enrichWithCloudflare(entries []*models.Entry, cfDetails map[string]api.CloudflareIngressEntry, cfDNSRecords map[string]string) {
	if cfDetails == nil {
		return
	}
	hostIndex := make(map[string]int, len(entries))
	for i, e := range entries {
		hostIndex[e.Hostname] = i
	}
	for hostname, cfEntry := range cfDetails {
		_, hasDNSRecord := cfDNSRecords[hostname]
		cfStatus := models.CloudflareStatus{
			Configured:       true,
			TunnelName:       cfEntry.TunnelName,
			TunnelID:         cfEntry.TunnelID,
			Service:          cfEntry.Service,
			Path:             cfEntry.Path,
			IsDefaultTunnel:  cfEntry.IsDefaultTunnel,
			HTTPHostHeader:   cfEntry.HTTPHostHeader,
			OriginServerName: cfEntry.OriginServerName,
			NoTLSVerify:      cfEntry.NoTLSVerify,
			Http2Origin:      cfEntry.Http2Origin,
			HasAccessPolicy:  cfEntry.HasAccessPolicy,
			HasDNSRecord:     hasDNSRecord,
		}
		if idx, ok := hostIndex[hostname]; ok {
			entries[idx].CloudflareStatus = cfStatus
		} else {
			entries = append(entries, &models.Entry{
				Hostname:         hostname,
				DataSource:       "CloudFlare",
				CloudflareStatus: cfStatus,
			})
		}
	}
}

// logLoadSummary logs notable findings after all data is loaded and merged.
func (d *DataLoader) logLoadSummary(entries []*models.Entry) {
	var (
		outOfSync  []string
		caddyOnly  []string
		stale      []string
		cfNoCNAME  []string
		cfNoHeader []string
	)

	for _, e := range entries {
		switch e.OverallStatus {
		case models.OutOfSync:
			outOfSync = append(outOfSync, e.Hostname)
		case models.CaddyOnly:
			caddyOnly = append(caddyOnly, e.Hostname)
		case models.Stale:
			stale = append(stale, e.Hostname)
		}
		if e.CloudflareStatus.Configured && !e.CloudflareStatus.HasDNSRecord {
			cfNoCNAME = append(cfNoCNAME, e.Hostname)
		}
		if e.CloudflareStatus.Configured && e.CloudflareStatus.HTTPHostHeader == "" {
			cfNoHeader = append(cfNoHeader, e.Hostname)
		}
	}

	if len(outOfSync) > 0 {
		logging.Warn("DNS out of sync", "count", len(outOfSync), "hostnames", outOfSync)
	}
	if len(caddyOnly) > 0 {
		logging.Warn("Caddy-only entries (not in DNS)", "count", len(caddyOnly), "hostnames", caddyOnly)
	}
	if len(stale) > 0 {
		logging.Warn("Stale DNS entries (not in Caddy)", "count", len(stale), "hostnames", stale)
	}
	if len(cfNoCNAME) > 0 {
		logging.Warn("CF tunnel rules missing DNS CNAME record", "count", len(cfNoCNAME), "hostnames", cfNoCNAME)
	}
	if len(cfNoHeader) > 0 {
		logging.Warn("CF tunnel rules missing HTTPHostHeader", "count", len(cfNoHeader), "hostnames", cfNoHeader)
	}
	if len(outOfSync)+len(caddyOnly)+len(stale)+len(cfNoCNAME)+len(cfNoHeader) == 0 {
		logging.Info("Load summary: all entries in sync, no issues found")
	}
}

func loadReportServices() []ServiceName {
	return []ServiceName{
		ServiceCaddy,
		ServiceUnbound,
		ServiceAdguard,
		ServiceDHCP,
		ServiceCloudflare,
		ServiceDNS,
	}
}

func newLoadReport() LoadReport {
	services := map[ServiceName]ServiceReport{
		ServiceCaddy:      {Status: ServicePending},
		ServiceUnbound:    {Status: ServicePending},
		ServiceAdguard:    {Status: ServicePending},
		ServiceDHCP:       {Status: ServicePending},
		ServiceCloudflare: {Status: ServicePending},
		ServiceDNS:        {Status: ServicePending},
	}
	return LoadReport{Services: services}
}

func (r LoadReport) set(service ServiceName, serviceReport ServiceReport) {
	r.Services[service] = serviceReport
}

func (r LoadReport) markUnfinished(status ServiceState, err string) {
	for _, service := range loadReportServices() {
		if r.Services[service].Status == ServicePending {
			r.set(service, ServiceReport{Status: status, Error: err})
		}
	}
}

func (d *DataLoader) contextErr() error {
	if d.ctx == nil {
		return nil
	}
	return d.ctx.Err()
}

func (d *DataLoader) emitReports(report LoadReport, services []ServiceName) {
	for _, service := range services {
		d.emitServiceReport(service, report.Services[service])
	}
}

func (d *DataLoader) emitAllReports(report LoadReport) {
	d.emitReports(report, loadReportServices())
}

func (d *DataLoader) emitServiceReport(service ServiceName, report ServiceReport) {
	d.emit(ProgressEvent{
		Service: service,
		Status:  report.Status,
		Count:   report.Count,
		Error:   report.Error,
	})
}

func (d *DataLoader) emit(event ProgressEvent) {
	if d.progress != nil {
		d.progress(event)
	}
}

func serviceReport[T any](items map[string]T, err error, skipped bool) ServiceReport {
	if skipped {
		return ServiceReport{Status: ServiceSkipped}
	}
	if err != nil {
		return ServiceReport{Status: ServiceFailed, Error: err.Error()}
	}
	return ServiceReport{Status: ServiceLoaded, Count: len(items)}
}

func countUniqueDHCPLeases(leases map[string]*api.DNSMasqLease) int {
	seen := make(map[*api.DNSMasqLease]bool)
	for _, lease := range leases {
		if lease != nil {
			seen[lease] = true
		}
	}
	return len(seen)
}

// resolveAllDNS resolves DNS for all entries in parallel using a worker pool.
func (d *DataLoader) resolveAllDNS(entries []*models.Entry) {
	const workers = 20
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, e := range entries {
		e := e
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			e.DNSResolved = d.resolveDNS(e.Hostname)
		}()
	}
	wg.Wait()
}

// loadCaddyHostnames loads full route details (hostname -> CaddyRouteInfo) from Caddy.
func (d *DataLoader) loadCaddyHostnames() (map[string]models.CaddyRouteInfo, error) {
	if d.caddyClient == nil {
		return nil, fmt.Errorf("Caddy client not initialized")
	}
	return d.caddyClient.GetHostnameDetails()
}

// loadUnboundOverrides loads DNS overrides from Unbound
func (d *DataLoader) loadUnboundOverrides() (map[string]*api.DNSOverride, error) {
	if d.unboundClient == nil {
		return nil, fmt.Errorf("Unbound client not initialized")
	}

	overrides, err := d.unboundClient.GetOverrides()
	if err != nil {
		return nil, err
	}

	// Index by hostname
	overrideMap := make(map[string]*api.DNSOverride)
	for i := range overrides {
		hostname := overrides[i].Host + "." + overrides[i].Domain
		overrideMap[hostname] = &overrides[i]
	}

	return overrideMap, nil
}

// loadAdguardRewrites loads DNS rewrites from AdGuard
func (d *DataLoader) loadAdguardRewrites() (map[string]*api.Rewrite, error) {
	if d.adguardClient == nil {
		return nil, fmt.Errorf("AdGuard client not initialized")
	}

	rewrites, err := d.adguardClient.ListRewrites()
	if err != nil {
		return nil, err
	}

	// Index by domain
	rewriteMap := make(map[string]*api.Rewrite)
	for i := range rewrites {
		rewriteMap[rewrites[i].Domain] = &rewrites[i]
	}

	return rewriteMap, nil
}

// loadDHCPLeases loads DHCP leases from DNSMasq.
// Returns a map keyed by BOTH short hostname (when present) AND IP address so
// that static reservations without a hostname can still be matched by IP.
func (d *DataLoader) loadDHCPLeases() (map[string]*api.DNSMasqLease, error) {
	if d.dnsmasqClient == nil {
		return nil, fmt.Errorf("DNSMasq client not initialized")
	}

	leases, err := d.dnsmasqClient.GetLeases()
	if err != nil {
		return nil, err
	}

	leaseMap := make(map[string]*api.DNSMasqLease)
	for i := range leases {
		// Index by short hostname (primary lookup path)
		if leases[i].Hostname != "" {
			leaseMap[leases[i].Hostname] = &leases[i]
		}
		// Also index by IP address so we can fall back when hostname is empty
		// (common for static DHCP reservations in OPNSense)
		if leases[i].IPAddress != "" {
			leaseMap[leases[i].IPAddress] = &leases[i]
		}
	}

	return leaseMap, nil
}

// buildEntries builds unified Entry models from all data sources
func (d *DataLoader) buildEntries(
	caddyHostnames map[string]models.CaddyRouteInfo,
	unboundOverrides map[string]*api.DNSOverride,
	adguardRewrites map[string]*api.Rewrite,
	dhcpLeases map[string]*api.DNSMasqLease,
) []*models.Entry {
	// Collect all unique hostnames from all sources
	hostnameSet := make(map[string]bool)

	// Add Caddy hostnames
	for hostname := range caddyHostnames {
		hostnameSet[hostname] = true
	}

	// Add Unbound hostnames
	for hostname := range unboundOverrides {
		hostnameSet[hostname] = true
	}

	// Add AdGuard hostnames
	for hostname := range adguardRewrites {
		hostnameSet[hostname] = true
	}

	// Build entries
	entries := make([]*models.Entry, 0, len(hostnameSet))

	for hostname := range hostnameSet {
		entry := d.buildEntry(hostname, caddyHostnames, unboundOverrides, adguardRewrites, dhcpLeases)
		entries = append(entries, entry)
	}

	return entries
}

// buildEntry builds a single Entry model for a hostname
func (d *DataLoader) buildEntry(
	hostname string,
	caddyHostnames map[string]models.CaddyRouteInfo,
	unboundOverrides map[string]*api.DNSOverride,
	adguardRewrites map[string]*api.Rewrite,
	dhcpLeases map[string]*api.DNSMasqLease,
) *models.Entry {
	entry := &models.Entry{
		Hostname: hostname,
	}

	// Caddy data (source of truth)
	if routeInfo, exists := caddyHostnames[hostname]; exists {
		entry.CaddyUpstream = routeInfo.Upstream
		entry.CaddyRoute = routeInfo
		entry.DataSource = "Caddy"

		// Extract IP and port from upstream
		// Upstream format: "10.0.0.112:8096" or "10.0.0.112"
		if parts := strings.Split(routeInfo.Upstream, ":"); len(parts) >= 1 {
			entry.CaddyIP = parts[0]
			if len(parts) == 2 {
				entry.CaddyPort = parts[1]
			}
		}
	}

	// Unbound data
	if override, exists := unboundOverrides[hostname]; exists {
		configured := override.Server != ""
		inSync := configured && override.Server == d.caddyServerIP
		entry.UnboundStatus = models.NewServiceStatus(configured, override.Server, inSync)

		if entry.DataSource == "" {
			entry.DataSource = "Unbound"
		}
	} else {
		entry.UnboundStatus = models.NotConfigured()
	}

	// AdGuard data
	if rewrite, exists := adguardRewrites[hostname]; exists {
		configured := rewrite.Answer != ""
		inSync := configured && rewrite.Answer == d.caddyServerIP
		entry.AdguardStatus = models.NewServiceStatus(configured, rewrite.Answer, inSync)

		if entry.DataSource == "" {
			entry.DataSource = "AdGuard"
		}
	} else {
		entry.AdguardStatus = models.NotConfigured()
	}

	// DHCP data
	// Try to find DHCP lease by: (1) short hostname, (2) IP from Caddy upstream.
	// Static reservations in OPNSense often have no hostname, only an IP.
	baseHostname := hostname
	if dotIdx := strings.Index(hostname, "."); dotIdx > 0 {
		baseHostname = hostname[:dotIdx]
	}
	var dhcpLease *api.DNSMasqLease
	if l, ok := dhcpLeases[baseHostname]; ok {
		dhcpLease = l
	} else if entry.CaddyIP != "" {
		dhcpLease = dhcpLeases[entry.CaddyIP] // may be nil — that's fine
	}

	if lease := dhcpLease; lease != nil {
		leaseType := "dynamic"
		if lease.Type == "static" {
			leaseType = "static"
		}

		inSync := lease.IPAddress == entry.CaddyIP
		entry.DHCPStatus = models.NewDHCPStatus(
			true,
			leaseType,
			lease.IPAddress,
			lease.MACAddress,
			lease.Hostname,
			inSync,
		)

		if entry.DataSource == "" {
			entry.DataSource = "DHCP"
		}
	} else {
		entry.DHCPStatus = models.NoDHCP()
	}

	// Store the Caddy server IP so HasDNSMismatch() can compare against it.
	entry.CaddyServerIP = d.caddyServerIP

	// Compute overall sync status (DNS resolution happens separately in parallel)
	entry.OverallStatus = models.ComputeSyncStatus(entry, d.caddyServerIP)

	return entry
}

// resolveDNS performs a DNS lookup for the hostname via Unbound directly (if available),
// falling back to the system resolver. This avoids Tailscale MagicDNS intercepting queries
// and returning Tailscale IPs instead of the LAN IPs that Unbound serves.
func (d *DataLoader) resolveDNS(hostname string) string {
	parent := d.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()

	resolver := net.DefaultResolver
	if d.unboundClient != nil {
		host := d.unboundClient.Host()
		if host != "" {
			resolver = &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "udp", host+":53")
				},
			}
		}
	}

	addrs, err := resolver.LookupHost(ctx, hostname)
	if err != nil {
		return "FAIL"
	}

	if len(addrs) == 0 {
		return "NONE"
	}

	return addrs[0]
}
