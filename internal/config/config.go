package config

import (
	"flag"
	"time"
)

type Config struct {
	URL           string
	Headless      bool
	Debug         bool
	GlobalTimeout time.Duration // Overall timeout
	ActionTimeout time.Duration // Timeout for individual actions
}

func Parse() *Config {
	cfg := &Config{}

	// Define flags
	flag.StringVar(&cfg.URL, "url", "", "Website URL to scrape (required)")
	flag.BoolVar(&cfg.Headless, "headless", false, "Run in headless mode")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode")

	// Timeout flags (in minutes)
	globalTimeout := flag.Int("timeout", 30, "Global timeout in minutes")
	actionTimeout := flag.Int("action-timeout", 1, "Individual action timeout in minutes")

	flag.Parse()

	// Convert timeouts to time.Duration
	cfg.GlobalTimeout = time.Duration(*globalTimeout) * time.Minute
	cfg.ActionTimeout = time.Duration(*actionTimeout) * time.Minute

	return cfg
}
