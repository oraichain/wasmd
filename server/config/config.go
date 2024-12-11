package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	errorsmod "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// DefaultIndexerServiceAddress is the default address the Indexer RPC server binds to.
	DefaultIndexerServiceAddress = "127.0.0.1:5050"

	// DefaultIServiceMetricsAddress is the default address the Indexer RPC Metrics server binds to.
	DefaultIServiceMetricsAddress = "127.0.0.1:5051"

	DefaultHTTPTimeout = 30 * time.Second

	DefaultHTTPIdleTimeout = 120 * time.Second

	// DefaultMaxOpenConnections represents the amount of open connections (unlimited = 0)
	DefaultMaxOpenConnections = 0

	IndexerFileName = "indexer"
	ConfigFileName  = "config"
)

// Config defines the server's top level configuration. It includes the default app config
// from the SDK as well as the RPC configuration to enable the RPC APIs.
type Config struct {
	IService IServiceConfig `mapstructure:"indexer-service"`
}

// IServiceConfig defines configuration for the RPC server.
type IServiceConfig struct {
	// Address defines the HTTP server to listen on
	Address string `mapstructure:"address"`
	// Enable defines if the RPC server should be enabled.
	Enable bool `mapstructure:"enable"`
	// EnableUnsafeCORS defines if CORS should be enabled (unsafe - use it at your own risk)
	EnableUnsafeCORS bool `mapstructure:"enabled-unsafe-cors"`
	// HTTPTimeout is the read/write timeout of http RPC server.
	HTTPTimeout time.Duration `mapstructure:"http-timeout"`
	// HTTPIdleTimeout is the idle timeout of http RPC server.
	HTTPIdleTimeout time.Duration `mapstructure:"http-idle-timeout"`
	// MaxOpenConnections sets the maximum number of simultaneous connections
	// for the server listener.
	MaxOpenConnections int `mapstructure:"max-open-connections"`
	// MetricsAddress defines the metrics server to listen on
	MetricsAddress string `mapstructure:"metrics-address"`
}

// DefaultConfig returns server's default configuration.
func DefaultConfig() *Config {
	return &Config{
		IService: *DefaultIServiceConfig(),
	}
}

// DefaultIServiceConfig returns an RPC config with the RPC API enabled by default
func DefaultIServiceConfig() *IServiceConfig {
	return &IServiceConfig{
		Enable:             true,
		Address:            DefaultIndexerServiceAddress,
		HTTPTimeout:        DefaultHTTPTimeout,
		HTTPIdleTimeout:    DefaultHTTPIdleTimeout,
		MaxOpenConnections: DefaultMaxOpenConnections,
		MetricsAddress:     DefaultIServiceMetricsAddress,
	}
}

// Validate returns an error if the RPC configuration fields are invalid.
func (c IServiceConfig) Validate() error {
	if c.HTTPTimeout < 0 {
		return errors.New("Indexer-RPC HTTP timeout duration cannot be negative")
	}

	if c.HTTPIdleTimeout < 0 {
		return errors.New("Indexer-RPC HTTP idle timeout duration cannot be negative")
	}

	return nil
}

// GetConfig returns a fully parsed Config object.
func GetConfig(v *viper.Viper) (Config, error) {
	return Config{
		IService: IServiceConfig{
			Enable:             v.GetBool("indexer-service.enable"),
			Address:            v.GetString("indexer-service.address"),
			HTTPTimeout:        v.GetDuration("indexer-service.http-timeout"),
			HTTPIdleTimeout:    v.GetDuration("indexer-service.http-idle-timeout"),
			MaxOpenConnections: v.GetInt("indexer-service.max-open-connections"),
			MetricsAddress:     v.GetString("indexer-service.metrics-address"),
		},
	}, nil
}

// ParseConfig retrieves the default environment configuration for the
// application.
func ParseConfig(v *viper.Viper) (*Config, error) {
	conf := DefaultConfig()
	err := v.Unmarshal(conf)

	return conf, err
}

// ValidateBasic returns an error any of the application configuration fields are invalid
func (c Config) ValidateBasic() error {

	if err := c.IService.Validate(); err != nil {
		return errorsmod.Wrapf(errortypes.ErrAppConfig, "invalid ethermint json-rpc config value: %s", err.Error())
	}

	return nil
}
