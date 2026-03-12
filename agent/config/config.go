package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the agent configuration
type Config struct {
	Controller ControllerConfig `yaml:"controller"`
	Agent      AgentConfig      `yaml:"agent"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// ControllerConfig holds controller connection settings
type ControllerConfig struct {
	Address    string     `yaml:"address"`
	MTLSConfig MTLSConfig `yaml:"mtls"`
}

// MTLSConfig holds mTLS certificate paths
type MTLSConfig struct {
	CACert     string `yaml:"ca_cert"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
}

// AgentConfig holds agent-specific settings
type AgentConfig struct {
	Hostname  string `yaml:"hostname"`
	TunName   string `yaml:"tun_name"`
	StateFile string `yaml:"state_file"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate required fields
	if config.Controller.Address == "" {
		return nil, fmt.Errorf("controller.address is required")
	}
	if config.Controller.MTLSConfig.CACert == "" {
		return nil, fmt.Errorf("controller.mtls.ca_cert is required")
	}
	if config.Controller.MTLSConfig.ClientCert == "" {
		return nil, fmt.Errorf("controller.mtls.client_cert is required")
	}
	if config.Controller.MTLSConfig.ClientKey == "" {
		return nil, fmt.Errorf("controller.mtls.client_key is required")
	}

	// Set defaults
	if config.Agent.TunName == "" {
		config.Agent.TunName = "ztna0"
	}
	if config.Agent.StateFile == "" {
		config.Agent.StateFile = "/var/lib/ztna/agent-state.json"
	}

	return &config, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	return &Config{
		Controller: ControllerConfig{
			Address: "controller.local:50051",
			MTLSConfig: MTLSConfig{
				CACert:     "/etc/ztna/ca.crt",
				ClientCert: "/etc/ztna/agent.crt",
				ClientKey:  "/etc/ztna/agent.key",
			},
		},
		Agent: AgentConfig{
			Hostname:  hostname,
			TunName:   "ztna0",
			StateFile: "/var/lib/ztna/agent-state.json",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Output: "stdout",
		},
	}
}
