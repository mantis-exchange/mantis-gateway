package config

import "os"

// Config holds all gateway configuration values, loaded from environment variables.
type Config struct {
	Port               string
	MatchingEngineAddr string
	JWTSecret          string
	CORSOrigins        string
}

// Load reads configuration from environment variables, falling back to defaults.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		MatchingEngineAddr: getEnv("MATCHING_ENGINE_ADDR", "localhost:9090"),
		JWTSecret:          getEnv("JWT_SECRET", "changeme"),
		CORSOrigins:        getEnv("CORS_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
