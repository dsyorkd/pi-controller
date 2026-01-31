package defaults

// Certificate Authority defaults
const (
	// CABackend is the default CA backend type
	CABackend = "local"
)

// Local CA defaults
const (
	// LocalCADataDir is the default CA data directory
	LocalCADataDir = "/etc/pi-controller/ca"

	// LocalCAValidityPeriod is the default CA certificate validity (10 years)
	LocalCAValidityPeriod = "87600h"

	// LocalCACertValidityPeriod is the default issued certificate validity (1 year)
	LocalCACertValidityPeriod = "8760h"

	// LocalCAKeySize is the default RSA key size
	LocalCAKeySize = 2048

	// LocalCAOrganization is the default organization name
	LocalCAOrganization = "Pi Controller"

	// LocalCAOrganizationalUnit is the default organizational unit
	LocalCAOrganizationalUnit = "Infrastructure"

	// LocalCACountry is the default country
	LocalCACountry = "US"

	// LocalCAProvince is the default province/state
	LocalCAProvince = "CA"

	// LocalCALocality is the default locality/city
	LocalCALocality = "San Francisco"
)

// Vault CA defaults
const (
	// VaultCAAddress is the default Vault server address
	VaultCAAddress = "https://vault.example.com:8200"

	// VaultCAMountPath is the default PKI mount path
	VaultCAMountPath = "pki"

	// VaultCATimeout is the default Vault connection timeout
	VaultCATimeout = "30s"

	// VaultCACertRole is the default certificate role
	VaultCACertRole = "pi-controller"

	// VaultCAAllowInsecure indicates whether insecure connections are allowed
	VaultCAAllowInsecure = false

	// VaultCAInsecureSkipVerify indicates whether to skip TLS verification
	VaultCAInsecureSkipVerify = false
)

// SSH defaults for CA operations
const (
	// SSHUser is the default SSH user
	SSHUser = "pi"

	// SSHPort is the default SSH port
	SSHPort = 22

	// SSHTimeout is the default SSH connection timeout
	SSHTimeout = "30s"

	// SSHStrictHostKeyChecking indicates whether strict host key checking is enabled
	SSHStrictHostKeyChecking = true

	// SSHKnownHostsFile is the default known hosts file path
	SSHKnownHostsFile = "/etc/pi-controller/known_hosts"
)

// Certificate config defaults
const (
	// CertDefaultValidityPeriod is the default certificate validity (1 year)
	CertDefaultValidityPeriod = "8760h"

	// CertRenewalThreshold is when to renew certificates (30 days before expiry)
	CertRenewalThreshold = "720h"

	// CertAllowWildcardDNS indicates whether wildcard DNS names are allowed
	CertAllowWildcardDNS = false

	// CertStoragePath is the default certificate storage path
	CertStoragePath = "./data/certificates"

	// CertCleanupInterval is the default cleanup interval
	CertCleanupInterval = "24h"

	// CertRetentionPeriod is how long to keep expired certificates (90 days)
	CertRetentionPeriod = "2160h"
)

// CertDefaultKeyUsage contains default key usage values
var CertDefaultKeyUsage = []string{
	"digital_signature",
	"key_encipherment",
}

// CertDefaultExtKeyUsage contains default extended key usage values
var CertDefaultExtKeyUsage = []string{
	"server_auth",
	"client_auth",
}

// CertAllowedDomains contains default allowed domains for certificates
var CertAllowedDomains = []string{
	"*.pi-controller.local",
	"*.cluster.local",
}
