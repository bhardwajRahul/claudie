package loadbalancer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/berops/claudie/internal/checksum"
	comm "github.com/berops/claudie/internal/command"
	"github.com/berops/claudie/internal/utils"
	"github.com/berops/claudie/proto/pb/spec"
	cluster_builder "github.com/berops/claudie/services/terraformer/server/domain/utils/cluster-builder"
	"github.com/berops/claudie/services/terraformer/server/domain/utils/templates"
	"github.com/berops/claudie/services/terraformer/server/domain/utils/terraform"
	"github.com/rs/zerolog"

	"google.golang.org/protobuf/proto"
)

const (
	TemplatesRootDir = "services/terraformer/templates"
)

type DNS struct {
	ProjectName string
	ClusterName string
	ClusterHash string

	DesiredNodeIPs []string
	CurrentNodeIPs []string

	CurrentDNS *spec.DNS
	DesiredDNS *spec.DNS

	// SpawnProcessLimit represents a synchronization channel which limits the number of spawned terraform
	// processes. This values should always be non-nil and be buffered, where the capacity indicates
	// the limit.
	SpawnProcessLimit chan struct{}
}

// CreateDNSRecords creates DNS records for the Loadbalancer cluster.
func (d DNS) CreateDNSRecords(logger zerolog.Logger) (string, error) {
	sublogger := logger.With().Str("endpoint", d.DesiredDNS.Endpoint).Logger()

	clusterID := fmt.Sprintf("%s-%s", d.ClusterName, d.ClusterHash)
	dnsID := fmt.Sprintf("%s-dns", clusterID)
	dnsDir := filepath.Join(cluster_builder.Output, dnsID)

	terraform := terraform.Terraform{
		Directory:         dnsDir,
		SpawnProcessLimit: d.SpawnProcessLimit,
	}

	terraform.Stdout = comm.GetStdOut(clusterID)
	terraform.Stderr = comm.GetStdErr(clusterID)

	defer func() {
		if err := os.RemoveAll(dnsDir); err != nil {
			sublogger.Err(err).Msgf("error while removing files in dir %q: %v", dnsDir, err)
		}
	}()

	if changedDNSProvider(d.CurrentDNS, d.DesiredDNS) {
		sublogger.Info().Msg("Destroying old DNS records")
		if err := d.generateFiles(dnsID, dnsDir, d.CurrentDNS, d.CurrentNodeIPs); err != nil {
			return "", fmt.Errorf("error while creating current state dns.tf files for %s : %w", dnsID, err)
		}
		// Delete the ones from desired state, in case this is a re-execution.
		if err := d.generateFiles(dnsID, dnsDir, d.DesiredDNS, d.DesiredNodeIPs); err != nil {
			return "", fmt.Errorf("error while creating desired state dns.tf files for %s : %w", dnsID, err)
		}
		if err := terraform.Init(); err != nil {
			return "", err
		}
		if err := terraform.Destroy(); err != nil {
			return "", err
		}

		if err := os.RemoveAll(dnsDir); err != nil {
			return "", fmt.Errorf("error while removing files in dir %q: %w", dnsDir, err)
		}

		sublogger.Info().Msg("Old DNS records were successfully destroyed")
	}

	sublogger.Info().Msg("Creating new DNS records")
	if err := d.generateFiles(dnsID, dnsDir, d.DesiredDNS, d.DesiredNodeIPs); err != nil {
		return "", fmt.Errorf("error while creating dns .tf files for %s : %w", dnsID, err)
	}
	if err := terraform.Init(); err != nil {
		return "", err
	}
	if err := terraform.Apply(); err != nil {
		return "", err
	}

	f := checksum.Digest128(filepath.Join(d.DesiredDNS.Provider.SpecName, templates.ExtractTargetPath(d.DesiredDNS.Provider.Templates)))
	k := fmt.Sprintf("%s_%s_%s", clusterID, d.DesiredDNS.GetProvider().GetSpecName(), hex.EncodeToString(f))

	output, err := terraform.Output(k)
	if err != nil {
		return "", fmt.Errorf("error while getting output from terraform for %s : %w", clusterID, err)
	}

	out, err := readDomain(output)
	if err != nil {
		return "", fmt.Errorf("error while reading output from terraform for %s : %w", clusterID, err)
	}

	outputID := fmt.Sprintf("%s-endpoint", clusterID)
	sublogger.Info().Msg("DNS records were successfully set up")

	return validateDomain(out.Domain[outputID]), nil
}

// DestroyDNSRecords destroys DNS records for the Loadbalancer cluster.
func (d DNS) DestroyDNSRecords(logger zerolog.Logger) error {
	sublogger := logger.With().Str("endpoint", d.CurrentDNS.Endpoint).Logger()

	sublogger.Info().Msg("Destroying DNS records")
	dnsID := fmt.Sprintf("%s-%s-dns", d.ClusterName, d.ClusterHash)
	dnsDir := filepath.Join(cluster_builder.Output, dnsID)

	defer func() {
		if err := os.RemoveAll(dnsDir); err != nil {
			sublogger.Err(err).Msgf("error while removing files in dir %q: %v", dnsDir, err)
		}
	}()

	if err := d.generateFiles(dnsID, dnsDir, d.CurrentDNS, d.CurrentNodeIPs); err != nil {
		return fmt.Errorf("error while creating dns records for %s : %w", dnsID, err)
	}

	terraform := terraform.Terraform{
		Directory:         dnsDir,
		SpawnProcessLimit: d.SpawnProcessLimit,
	}

	terraform.Stdout = comm.GetStdOut(dnsID)
	terraform.Stderr = comm.GetStdErr(dnsID)

	if err := terraform.Init(); err != nil {
		return err
	}

	if err := terraform.Destroy(); err != nil {
		return err
	}

	sublogger.Info().Msg("DNS records were successfully destroyed")

	return nil
}

// generateFiles creates all the necessary terraform files used to create/destroy DNS.
func (d DNS) generateFiles(dnsID, dnsDir string, dns *spec.DNS, nodeIPs []string) error {
	backend := templates.Backend{
		ProjectName: d.ProjectName,
		ClusterName: dnsID,
		Directory:   dnsDir,
	}

	if err := backend.CreateTFFile(); err != nil {
		return err
	}

	usedProviders := templates.UsedProviders{
		ProjectName: d.ProjectName,
		ClusterName: dnsID,
		Directory:   dnsDir,
	}

	if err := usedProviders.CreateUsedProviderDNS(dns); err != nil {
		return err
	}

	templateDir := filepath.Join(TemplatesRootDir, dnsID, dns.GetProvider().GetSpecName())
	if err := templates.DownloadProvider(templateDir, dns.GetProvider()); err != nil {
		return fmt.Errorf("failed to download templates for DNS %q: %w", dnsID, err)
	}

	path := templates.ExtractTargetPath(dns.Provider.Templates)

	g := templates.Generator{
		ID:                dnsID,
		TargetDirectory:   dnsDir,
		ReadFromDirectory: templateDir,
		TemplatePath:      path,
		Fingerprint:       hex.EncodeToString(checksum.Digest128(filepath.Join(dns.Provider.SpecName, path))),
	}

	data := templates.DNS{
		DNSZone:     dns.DnsZone,
		Hostname:    dns.Hostname,
		ClusterName: d.ClusterName,
		ClusterHash: d.ClusterHash,
		RecordData:  templates.RecordData{IP: templateIPData(nodeIPs)},
		Provider:    dns.Provider,
	}

	if err := g.GenerateDNS(&data); err != nil {
		return fmt.Errorf("failed to generate dns templates for %q: %w", dnsID, err)
	}

	if err := utils.CreateKeyFile(utils.GetAuthCredentials(data.Provider), g.TargetDirectory, data.Provider.SpecName); err != nil {
		return fmt.Errorf("error creating provider credential key file for provider %s in %s : %w", data.Provider.SpecName, g.TargetDirectory, err)
	}

	return nil
}

// validateDomain validates the domain does not start with ".".
func validateDomain(s string) string {
	if s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	return s
}

// readDomain reads full domain from terraform output.
func readDomain(data string) (templates.DNSDomain, error) {
	var result templates.DNSDomain
	err := json.Unmarshal([]byte(data), &result.Domain)
	return result, err
}

func changedDNSProvider(currentDNS, desiredDNS *spec.DNS) bool {
	// DNS not yet created
	if currentDNS == nil {
		return false
	}
	// DNS provider are same
	if currentDNS.Provider.SpecName == desiredDNS.Provider.SpecName {
		if proto.Equal(currentDNS.Provider, desiredDNS.Provider) {
			return false
		}
	}
	return true
}

func templateIPData(ips []string) []templates.IPData {
	out := make([]templates.IPData, 0, len(ips))

	for _, ip := range ips {
		out = append(out, templates.IPData{V4: ip})
	}

	return out
}
