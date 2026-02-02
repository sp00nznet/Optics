package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Scanning ScanningConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type StorageConfig struct {
	MountPath     string
	ThumbnailPath string
}

type ScanningConfig struct {
	Extensions     []string
	IgnorePatterns []string
	BatchSize      int
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/optics")

	// Environment variable overrides
	viper.SetEnvPrefix("OPTICS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "optics")
	viper.SetDefault("database.user", "optics")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("jwt.expiry", "24h")
	viper.SetDefault("storage.mount_path", "/mnt/3d-models")
	viper.SetDefault("storage.thumbnail_path", "/data/thumbnails")
	viper.SetDefault("scanning.extensions", []string{"stl", "scad", "obj", "3mf", "step", "stp"})
	viper.SetDefault("scanning.ignore_patterns", []string{"System Volume Information", "*.tmp"})
	viper.SetDefault("scanning.batch_size", 100)

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	expiry, err := time.ParseDuration(viper.GetString("jwt.expiry"))
	if err != nil {
		expiry = 24 * time.Hour
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetString("server.port"),
			Host: viper.GetString("server.host"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetInt("database.port"),
			Name:     viper.GetString("database.name"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			SSLMode:  viper.GetString("database.sslmode"),
		},
		JWT: JWTConfig{
			Secret: viper.GetString("jwt.secret"),
			Expiry: expiry,
		},
		Storage: StorageConfig{
			MountPath:     viper.GetString("storage.mount_path"),
			ThumbnailPath: viper.GetString("storage.thumbnail_path"),
		},
		Scanning: ScanningConfig{
			Extensions:     viper.GetStringSlice("scanning.extensions"),
			IgnorePatterns: viper.GetStringSlice("scanning.ignore_patterns"),
			BatchSize:      viper.GetInt("scanning.batch_size"),
		},
	}

	return cfg, nil
}
