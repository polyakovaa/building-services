package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type GatewayConfig struct {
	Server   ServerConfig             `yaml:"server"`
	Services map[string]ServiceConfig `yaml:"services"`
	Logging  LoggingConfig            `yaml:"logging"`
}

type ServerConfig struct {
	Port    string        `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type ServiceConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func LoadConfig(path string) (*GatewayConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config GatewayConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func (c *GatewayConfig) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service must be configured")
	}

	for name, service := range c.Services {
		if service.Address == "" {
			return fmt.Errorf("address for service %s is required", name)
		}
		if service.Timeout == 0 {
			return fmt.Errorf("timeout for service %s is required", name)
		}
	}

	return nil
}

func (c *GatewayConfig) GetServiceConfig(serviceName string) (*ServiceConfig, error) {
	service, exists := c.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not configured", serviceName)
	}

	if service.Timeout == 0 {
		service.Timeout = 5 * time.Second
	}

	return &service, nil
}
