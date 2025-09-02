package config

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	Server struct {
		Port            string        `yaml:"port" json:"port" default:":8080"`
		ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout" default:"10s"`
		WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout" default:"10s"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout" default:"30s"`
	} `yaml:"server" json:"server"`

	Metrics struct {
		CollectionInterval     time.Duration `yaml:"collection_interval" json:"collection_interval" default:"15s"`
		CommandTimeout         time.Duration `yaml:"command_timeout" json:"command_timeout" default:"10s"`
		EnableSystemMetrics    bool          `yaml:"enable_system_metrics" json:"enable_system_metrics" default:"true"`
		EnableContainerMetrics bool          `yaml:"enable_container_metrics" json:"enable_container_metrics" default:"true"`
		EnableNetworkMetrics   bool          `yaml:"enable_network_metrics" json:"enable_network_metrics" default:"true"`
	} `yaml:"metrics" json:"metrics"`

	Containers struct {
		DockerEnabled  bool     `yaml:"docker_enabled" json:"docker_enabled" default:"true"`
		PodmanEnabled  bool     `yaml:"podman_enabled" json:"podman_enabled" default:"true"`
		MonitoredNames []string `yaml:"monitored_names" json:"monitored_names"`
		IgnoredNames   []string `yaml:"ignored_names" json:"ignored_names"`
	} `yaml:"containers" json:"containers"`

	Network struct {
		PingTargets       []string `yaml:"ping_targets" json:"ping_targets"`
		MonitorLoopback   bool     `yaml:"monitor_loopback" json:"monitor_loopback" default:"false"`
		IgnoredInterfaces []string `yaml:"ignored_interfaces" json:"ignored_interfaces"`
	} `yaml:"network" json:"network"`

	Benchmarking struct {
		WorkloadsPath  string        `yaml:"workloads_path" json:"workloads_path" default:"./workloads"`
		ResultsPath    string        `yaml:"results_path" json:"results_path" default:"./results"`
		MaxConcurrency int           `yaml:"max_concurrency" json:"max_concurrency" default:"10"`
		TestDuration   time.Duration `yaml:"test_duration" json:"test_duration" default:"5m"`
	} `yaml:"benchmarking" json:"benchmarking"`

	Logging struct {
		Level  string `yaml:"level" json:"level" default:"info"`
		Format string `yaml:"format" json:"format" default:"json"`
	} `yaml:"logging" json:"logging"`
}

func New() *Config {
	config := &Config{}
	return config
}

// LoadFromJSON loads configuration from a JSON file
func LoadFromJSON(path string) (*Config, error) {
	// Create empty config
	config := &Config{}

	// Open the JSON file
	file, err := os.Open(path)
	if err != nil {
		return nil, err // Fail if file doesn't exist
	}
	defer file.Close()

	// Decode JSON into config struct
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields() // Fail on unknown fields

	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
