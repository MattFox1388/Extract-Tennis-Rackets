package main

import (
	"log"

	"extract-app/internal/browser"
	"extract-app/internal/config"
	"extract-app/internal/scraper"
)

func main() {
	// Parse config
	cfg := config.Parse()

	log.Println("Initializing browser...")

	// Initialize browser
	ctx, cancel := browser.NewChrome(cfg)
	if ctx == nil {
		log.Fatal("Failed to initialize browser")
		return
	}

	// Ensure cleanup
	defer func() {
		log.Println("Cleaning up browser...")
		cancel()
	}()

	log.Println("Browser initialized, creating scraper...")

	// Create scraper
	scraper := scraper.New(ctx, cfg)

	log.Println("Starting scraping process...")

	// Run scraper
	_, err := scraper.GetOptions(cfg.URL)
	if err != nil {
		log.Fatalf("Scraping failed: %v", err)
	}

	log.Println("Scraping completed successfully")
}
