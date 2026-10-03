package hub

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	envMaxConnections        = "WS_MAX_CONNECTIONS"
	envMaxRoomsPerConnection = "WS_MAX_ROOMS_PER_CONNECTION"
	envClientQueueSize       = "WS_CLIENT_QUEUE_SIZE"
	envMaxFrameBytes         = "WS_MAX_FRAME_BYTES"
	envPingInterval          = "WS_PING_INTERVAL"
	envWriteTimeout          = "WS_WRITE_TIMEOUT"
	envShutdownTimeout       = "WS_SHUTDOWN_TIMEOUT"
)

type Config struct {
	// MaxConnections caps concurrent sockets, bounding memory per process.
	MaxConnections int
	// MaxRoomsPerConnection caps how many rooms one socket may subscribe to.
	MaxRoomsPerConnection int
	// ClientQueueSize is the per-connection outbound backlog
	ClientQueueSize int
	// MaxFrameBytes bounds inbound client frames
	MaxFrameBytes int64
	// how often writer pings
	PingInterval time.Duration
	// WriteTimeout bounds a single outbound write or ping.
	WriteTimeout time.Duration
	// ShutdownTimeout bounds how long Shutdown waits for writers to drain.
	ShutdownTimeout time.Duration
}

func defaultConfig() Config {
	return Config{
		MaxConnections:        1000,
		MaxRoomsPerConnection: 50,
		ClientQueueSize:       64,
		MaxFrameBytes:         4096,
		PingInterval:          30 * time.Second,
		WriteTimeout:          10 * time.Second,
		ShutdownTimeout:       5 * time.Second,
	}
}

func ConfigFromEnv() (Config, error) {
	cfg := defaultConfig()
	var err error

	if cfg.MaxConnections, err = envInt(envMaxConnections, cfg.MaxConnections); err != nil {
		return Config{}, err
	}
	if cfg.MaxRoomsPerConnection, err = envInt(envMaxRoomsPerConnection, cfg.MaxRoomsPerConnection); err != nil {
		return Config{}, err
	}
	if cfg.ClientQueueSize, err = envInt(envClientQueueSize, cfg.ClientQueueSize); err != nil {
		return Config{}, err
	}
	if cfg.MaxFrameBytes, err = envInt64(envMaxFrameBytes, cfg.MaxFrameBytes); err != nil {
		return Config{}, err
	}
	if cfg.PingInterval, err = envDuration(envPingInterval, cfg.PingInterval); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = envDuration(envWriteTimeout, cfg.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = envDuration(envShutdownTimeout, cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func envInt(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", name, raw)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0, got %d", name, parsed)
	}
	return parsed, nil
}

func envInt64(name string, fallback int64) (int64, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", name, raw)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0, got %d", name, parsed)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 30s or 1m, got %q", name, raw)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0, got %s", name, parsed)
	}
	return parsed, nil
}
