package client

import "time"

// Global option defaults, mirroring pybatfish.client.options.Options.
const (
	// CoordinatorHost is the default Batfish coordinator host.
	CoordinatorHost = "localhost"
	// CoordinatorWorkPort is the deprecated v1 coordinator work port.
	CoordinatorWorkPort = SvcCfgWorkPort
	// CoordinatorWorkV2Port is the v2 coordinator work port.
	CoordinatorWorkV2Port = SvcCfgWorkV2Port
	// UseSSL reports whether SSL is enabled by default.
	UseSSL = !SvcCfgWorkSSLDisable
	// VerifySSLCerts is true in production; only disable for local testing.
	VerifySSLCerts = true

	// DefaultNetworkPrefix is prepended to auto-generated network names.
	DefaultNetworkPrefix = "pcp"
	// DefaultQuestionPrefix is prepended to auto-generated question names.
	DefaultQuestionPrefix = "q"
	// DefaultSnapshotPrefix is prepended to auto-generated snapshot names.
	DefaultSnapshotPrefix = "ss_"

	// RequestBackoffFactor is the urllib3 Retry backoff factor.
	RequestBackoffFactor = 0.8
	// MaxInitialTriesToConnect is the number of initial connection tries.
	MaxInitialTriesToConnect = 3
	// MaxRetriesToConnect is the number of retries after session setup.
	MaxRetriesToConnect = 10

	// DefaultTimeout is the default per-request timeout.
	DefaultTimeout = 30 * time.Second
)
