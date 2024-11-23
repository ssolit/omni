package reth

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Version defines the geth version deployed to all networks.
const Version = "1.0.6"

// Duration is a custom type that wraps time.Duration to handle unmarshaling from TOML tables or strings.
type Duration struct {
	time.Duration
}

// UnmarshalTOML implements the toml.Unmarshaler interface for the Duration type.
// It can handle both string durations (e.g., "10m") and tables with 'secs' and 'nanos' keys.
func (d *Duration) UnmarshalTOML(data interface{}) error {
	switch value := data.(type) {
	case string:
		// Handle string duration (e.g., "10m")
		parsedDuration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration string: %v", err)
		}
		d.Duration = parsedDuration
		return nil
	case map[string]interface{}:
		// Handle table duration with 'secs' and 'nanos'
		secs, ok := value["secs"]
		if !ok {
			return fmt.Errorf("missing 'secs' in Duration")
		}

		nanos, ok := value["nanos"]
		if !ok {
			return fmt.Errorf("missing 'nanos' in Duration")
		}

		// Convert 'secs' to int64
		var secsInt int64
		switch v := secs.(type) {
		case int64:
			secsInt = v
		case int:
			secsInt = int64(v)
		default:
			return fmt.Errorf("invalid type for 'secs' in Duration")
		}

		// Convert 'nanos' to int64
		var nanosInt int64
		switch v := nanos.(type) {
		case int64:
			nanosInt = v
		case int:
			nanosInt = int64(v)
		default:
			return fmt.Errorf("invalid type for 'nanos' in Duration")
		}

		d.Duration = time.Duration(secsInt)*time.Second + time.Duration(nanosInt)
		return nil
	default:
		return fmt.Errorf("invalid type for Duration, expected string or map, got %T", data)
	}
}

// FullConfig defines the configuration structure for a Reth Ethereum node.
type FullConfig struct {
	Stages   StagesConfig   `toml:"stages"`
	Peers    PeersConfig    `toml:"peers"`
	Sessions SessionsConfig `toml:"sessions"`
}

// StagesConfig contains configurations for various stages of the Reth sync process.
type StagesConfig struct {
	Headers             HeadersConfig           `toml:"headers"`
	Bodies              BodiesConfig            `toml:"bodies"`
	SenderRecovery      SenderRecoveryConfig    `toml:"sender_recovery"`
	Execution           ExecutionConfig         `toml:"execution"`
	Prune               PruneConfig             `toml:"prune"`
	AccountHashing      HashingConfig           `toml:"account_hashing"`
	StorageHashing      HashingConfig           `toml:"storage_hashing"`
	Merkle              MerkleConfig            `toml:"merkle"`
	TransactionLookup   TransactionLookupConfig `toml:"transaction_lookup"`
	IndexAccountHistory IndexConfig             `toml:"index_account_history"`
	IndexStorageHistory IndexConfig             `toml:"index_storage_history"`
	ETL                 ETLConfig               `toml:"etl"`
}

type HeadersConfig struct {
	DownloaderMaxConcurrentRequests int `toml:"downloader_max_concurrent_requests"`
	DownloaderMinConcurrentRequests int `toml:"downloader_min_concurrent_requests"`
	DownloaderMaxBufferedResponses  int `toml:"downloader_max_buffered_responses"`
	DownloaderRequestLimit          int `toml:"downloader_request_limit"`
	CommitThreshold                 int `toml:"commit_threshold"`
}

type BodiesConfig struct {
	DownloaderRequestLimit               int   `toml:"downloader_request_limit"`
	DownloaderStreamBatchSize            int   `toml:"downloader_stream_batch_size"`
	DownloaderMaxBufferedBlocksSizeBytes int64 `toml:"downloader_max_buffered_blocks_size_bytes"`
	DownloaderMinConcurrentRequests      int   `toml:"downloader_min_concurrent_requests"`
	DownloaderMaxConcurrentRequests      int   `toml:"downloader_max_concurrent_requests"`
}

type SenderRecoveryConfig struct {
	CommitThreshold int `toml:"commit_threshold"`
}

type ExecutionConfig struct {
	MaxBlocks        int      `toml:"max_blocks"`
	MaxChanges       int      `toml:"max_changes"`
	MaxCumulativeGas int64    `toml:"max_cumulative_gas"`
	MaxDuration      Duration `toml:"max_duration"`
}

type PruneConfig struct {
	CommitThreshold int `toml:"commit_threshold"`
}

type HashingConfig struct {
	CleanThreshold  int `toml:"clean_threshold"`
	CommitThreshold int `toml:"commit_threshold"`
}

type MerkleConfig struct {
	CleanThreshold int `toml:"clean_threshold"`
}

type TransactionLookupConfig struct {
	ChunkSize int `toml:"chunk_size"`
}

type IndexConfig struct {
	CommitThreshold int `toml:"commit_threshold"`
}

type ETLConfig struct {
	FileSize int64 `toml:"file_size"`
}

// PeersConfig contains configurations related to peer management.
type PeersConfig struct {
	RefillSlotsInterval Duration                `toml:"refill_slots_interval"`
	TrustedNodes        []string                `toml:"trusted_nodes"`
	TrustedNodesOnly    bool                    `toml:"trusted_nodes_only"`
	MaxBackoffCount     int                     `toml:"max_backoff_count"`
	BanDuration         Duration                `toml:"ban_duration"`
	ConnectionInfo      ConnectionInfoConfig    `toml:"connection_info"`
	ReputationWeights   ReputationWeightsConfig `toml:"reputation_weights"`
	BackoffDurations    BackoffDurationsConfig  `toml:"backoff_durations"`
}

type ConnectionInfoConfig struct {
	MaxOutbound                int `toml:"max_outbound"`
	MaxInbound                 int `toml:"max_inbound"`
	MaxConcurrentOutboundDials int `toml:"max_concurrent_outbound_dials"`
}

type ReputationWeightsConfig struct {
	BadMessage              int `toml:"bad_message"`
	BadBlock                int `toml:"bad_block"`
	BadTransactions         int `toml:"bad_transactions"`
	AlreadySeenTransactions int `toml:"already_seen_transactions"`
	Timeout                 int `toml:"timeout"`
	BadProtocol             int `toml:"bad_protocol"`
	FailedToConnect         int `toml:"failed_to_connect"`
	Dropped                 int `toml:"dropped"`
	BadAnnouncement         int `toml:"bad_announcement"`
}

type BackoffDurationsConfig struct {
	Low    Duration `toml:"low"`
	Medium Duration `toml:"medium"`
	High   Duration `toml:"high"`
	Max    Duration `toml:"max"`
}

// SessionsConfig contains configurations for session management.
type SessionsConfig struct {
	SessionCommandBuffer          int          `toml:"session_command_buffer"`
	SessionEventBuffer            int          `toml:"session_event_buffer"`
	Limits                        LimitsConfig `toml:"limits"`
	InitialInternalRequestTimeout Duration     `toml:"initial_internal_request_timeout"`
	ProtocolBreachRequestTimeout  Duration     `toml:"protocol_breach_request_timeout"`
	PendingSessionTimeout         Duration     `toml:"pending_session_timeout"`
}

type LimitsConfig struct {
	// Define fields if any, or leave empty if not used
}

// defaultRethConfig returns the default reth config.
func defaultRethConfig() FullConfig {
	// File path to the config
	configFilePath := "testdata/default_config.toml"

	// Read the entire file content
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Create a new Config object
	var config FullConfig

	// Decode the TOML content into the Config struct
	if _, err := toml.Decode(string(content), &config); err != nil {
		log.Fatalf("Error decoding TOML: %v", err)
	}

	return config
}
