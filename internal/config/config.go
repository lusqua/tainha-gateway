package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type AuthConfig struct {
	Secret           string `yaml:"secret" mapstructure:"secret"`
	DefaultProtected bool   `yaml:"defaultProtected" mapstructure:"defaultProtected"`
	AuthService      string `yaml:"authService" mapstructure:"authService"`
	AuthPath         string `yaml:"authPath" mapstructure:"authPath"`
}

type RateLimitConfig struct {
	Enabled       bool `yaml:"enabled" mapstructure:"enabled"`
	RequestsPerSec int  `yaml:"requestsPerSec" mapstructure:"requestsPerSec"`
	Burst         int  `yaml:"burst" mapstructure:"burst"`
}

type TelemetryConfig struct {
	Enabled          bool   `yaml:"enabled" mapstructure:"enabled"`
	ServiceName      string `yaml:"serviceName" mapstructure:"serviceName"`
	ExporterEndpoint string `yaml:"exporterEndpoint" mapstructure:"exporterEndpoint"`
}

type MappingCacheConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	TTLSec  int  `yaml:"ttlSec" mapstructure:"ttlSec"`
	MaxSize int  `yaml:"maxSize" mapstructure:"maxSize"`
}

type CircuitBreakerConfig struct {
	Enabled          bool `yaml:"enabled" mapstructure:"enabled"`
	MaxFailures      int  `yaml:"maxFailures" mapstructure:"maxFailures"`
	TimeoutSec       int  `yaml:"timeoutSec" mapstructure:"timeoutSec"`
	HalfOpenRequests int  `yaml:"halfOpenRequests" mapstructure:"halfOpenRequests"`
}

type BaseConfig struct {
	Port            int                  `mapstructure:"port" default:"8080"`
	BasePath        string               `mapstructure:"basePath" default:"/api"`
	Auth            AuthConfig           `mapstructure:"auth"`
	RateLimit       RateLimitConfig      `mapstructure:"rateLimit"`
	Telemetry       TelemetryConfig      `mapstructure:"telemetry"`
	MappingCache    MappingCacheConfig   `mapstructure:"mappingCache" yaml:"mappingCache"`
	CircuitBreaker  CircuitBreakerConfig `mapstructure:"circuitBreaker" yaml:"circuitBreaker"`
	ReadTimeoutSec  int                  `mapstructure:"readTimeoutSec" yaml:"readTimeoutSec"`
	WriteTimeoutSec int                  `mapstructure:"writeTimeoutSec" yaml:"writeTimeoutSec"`
	IdleTimeoutSec  int                  `mapstructure:"idleTimeoutSec" yaml:"idleTimeoutSec"`
}

type RouteMapping struct {
	Path             string `mapstructure:"path"`
	Service          string `mapstructure:"service"`
	Tag              string `mapstructure:"tag"`
	RemoveKeyMapping bool   `mapstructure:"removeKeyMapping"`
}

type Route struct {
	Method      string         `mapstructure:"method"`
	Path        string         `mapstructure:"path"`
	Service     string         `mapstructure:"service"`
	Mapping     []RouteMapping `mapstructure:"mapping"`
	Route       string         `mapstructure:"route"`
	IsSSE       bool           `mapstructure:"isSSE,omitempty" yaml:"isSSE,omitempty"`
	IsWebSocket bool           `mapstructure:"isWebSocket,omitempty" yaml:"isWebSocket,omitempty"`
	Public      bool           `mapstructure:"public,omitempty" yaml:"public,omitempty"`
}

type Config struct {
	BaseConfig BaseConfig `mapstructure:"config"`
	Routes     []Route    `mapstructure:"routes"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	config.applyDefaults()

	return &config, nil
}

func (c *Config) applyDefaults() {
	if c.BaseConfig.Port == 0 {
		c.BaseConfig.Port = 8080
	}
	if c.BaseConfig.ReadTimeoutSec == 0 {
		c.BaseConfig.ReadTimeoutSec = 15
	}
	if c.BaseConfig.WriteTimeoutSec == 0 {
		c.BaseConfig.WriteTimeoutSec = 30
	}
	if c.BaseConfig.IdleTimeoutSec == 0 {
		c.BaseConfig.IdleTimeoutSec = 60
	}
	if c.BaseConfig.Telemetry.Enabled && c.BaseConfig.Telemetry.ServiceName == "" {
		c.BaseConfig.Telemetry.ServiceName = "tainha-gateway"
	}
	if c.BaseConfig.MappingCache.Enabled {
		if c.BaseConfig.MappingCache.TTLSec == 0 {
			c.BaseConfig.MappingCache.TTLSec = 60
		}
		if c.BaseConfig.MappingCache.MaxSize == 0 {
			c.BaseConfig.MappingCache.MaxSize = 1000
		}
	}
	if c.BaseConfig.CircuitBreaker.Enabled {
		if c.BaseConfig.CircuitBreaker.MaxFailures == 0 {
			c.BaseConfig.CircuitBreaker.MaxFailures = 5
		}
		if c.BaseConfig.CircuitBreaker.TimeoutSec == 0 {
			c.BaseConfig.CircuitBreaker.TimeoutSec = 30
		}
		if c.BaseConfig.CircuitBreaker.HalfOpenRequests == 0 {
			c.BaseConfig.CircuitBreaker.HalfOpenRequests = 1
		}
	}
	if c.BaseConfig.RateLimit.Enabled {
		if c.BaseConfig.RateLimit.RequestsPerSec == 0 {
			c.BaseConfig.RateLimit.RequestsPerSec = 100
		}
		if c.BaseConfig.RateLimit.Burst == 0 {
			c.BaseConfig.RateLimit.Burst = c.BaseConfig.RateLimit.RequestsPerSec * 2
		}
	}
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true,
}

func (c *Config) Validate() error {
	var errs []string

	if len(c.Routes) == 0 {
		errs = append(errs, "no routes defined")
	}

	if c.BaseConfig.Auth.DefaultProtected {
		if c.BaseConfig.Auth.Secret == "" && c.BaseConfig.Auth.AuthService == "" {
			errs = append(errs, "auth is defaultProtected but neither secret nor authService is configured")
		}
	}

	seen := make(map[string]bool)
	for i, route := range c.Routes {
		prefix := fmt.Sprintf("routes[%d]", i)

		if route.Method == "" {
			errs = append(errs, fmt.Sprintf("%s: method is required", prefix))
		} else if !validMethods[strings.ToUpper(route.Method)] {
			errs = append(errs, fmt.Sprintf("%s: invalid method %q", prefix, route.Method))
		}

		if route.Route == "" {
			errs = append(errs, fmt.Sprintf("%s: route is required", prefix))
		}
		if route.Service == "" {
			errs = append(errs, fmt.Sprintf("%s: service is required", prefix))
		}
		if route.Path == "" {
			errs = append(errs, fmt.Sprintf("%s: path is required", prefix))
		}

		key := route.Method + " " + route.Route
		if seen[key] {
			errs = append(errs, fmt.Sprintf("%s: duplicate route %s", prefix, key))
		}
		seen[key] = true

		for j, mapping := range route.Mapping {
			mPrefix := fmt.Sprintf("%s.mapping[%d]", prefix, j)
			if mapping.Path == "" {
				errs = append(errs, fmt.Sprintf("%s: path is required", mPrefix))
			}
			if mapping.Service == "" {
				errs = append(errs, fmt.Sprintf("%s: service is required", mPrefix))
			}
			if mapping.Tag == "" {
				errs = append(errs, fmt.Sprintf("%s: tag is required", mPrefix))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
