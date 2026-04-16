package config

import (
	"github.com/spf13/viper"
)

type AuthConfig struct {
	Secret           string `yaml:"secret"`
	DefaultProtected bool   `yaml:"defaultProtected"`
	AuthService      string `mapstructure:"authService" yaml:"authService"`
	AuthPath         string `mapstructure:"authPath" yaml:"authPath"`
}

type BaseConfig struct {
	Port     int        `mapstructure:"port" default:"8080"`
	BasePath string     `mapstructure:"basePath" default:"/api"`
	Auth     AuthConfig `mapstructure:"auth"`
}

type RouteMapping struct {
	Path             string `mapstructure:"path"`
	Service          string `mapstructure:"service"`
	Tag              string `mapstructure:"tag"`
	RemoveKeyMapping bool   `mapstructure:"removeKeyMapping"`
}

type Route struct {
	Method  string         `mapstructure:"method"`
	Path    string         `mapstructure:"path"`
	Service string         `mapstructure:"service"`
	Mapping []RouteMapping `mapstructure:"mapping"`
	Route   string         `mapstructure:"route"`
	IsSSE   bool           `mapstructure:"isSSE,omitempty" yaml:"isSSE,omitempty"`
	Public  bool           `mapstructure:"public,omitempty" yaml:"public,omitempty"`
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

	return &config, nil
}
