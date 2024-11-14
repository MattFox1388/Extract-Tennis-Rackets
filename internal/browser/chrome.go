package browser

import (
	"context"
	"log"

	"extract-app/internal/config"

	"github.com/chromedp/chromedp"
)

func NewChrome(cfg *config.Config) (context.Context, context.CancelFunc) {
	// Create base context
	baseCtx := context.Background()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,

		// Disable updates and popups
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-background-downloads", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-default-apps", true),

		// Basic settings
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", cfg.Headless),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.Flag("start-maximized", true),

		// Stability flags
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-component-extensions-with-background-pages", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("no-sandbox", true),
	)

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(baseCtx, opts...)

	// Create browser context with debug logging
	browserCtx, browserCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			log.Printf("CHROME: "+format, args...)
		}),
	)

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(browserCtx, cfg.GlobalTimeout)

	// Create combined cancel function
	cancelFunc := func() {
		log.Println("Canceling browser contexts...")
		timeoutCancel()
		browserCancel()
		allocCancel()
	}

	// Ensure browser is started
	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		log.Println("Starting new browser instance...")
		return nil
	})); err != nil {
		log.Printf("Failed to start browser: %v", err)
		cancelFunc()
		return nil, func() {}
	}

	return timeoutCtx, cancelFunc
}
