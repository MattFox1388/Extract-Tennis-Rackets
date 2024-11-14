package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"extract-app/internal/config"

	"github.com/chromedp/chromedp"
)

// Result represents the scraped data for each option
type Result struct {
	Option   string   // The option text that was selected
	RacNames []string // List of RAC names found for this option
}

type Scraper struct {
	ctx           context.Context
	actionTimeout time.Duration
	debug         bool
}

type RacquetSpecs struct {
	Name          string
	Error         string
	HeadSize      string
	Length        string
	Balance       string
	SwingWeight   string
	BeamWidth     string
	TipOrShaft    string
	Composition   string
	PowerLevel    string
	Stiffness     string
	StringPattern string
	MainSkip      string
	StringTension string
}

func (r *RacquetSpecs) Print() {
	fmt.Printf("Racquet Name: %s\n", r.Name)
	fmt.Printf("Error: %s\n", r.Error)
	fmt.Printf("Head Size: %s\n", r.HeadSize)
	fmt.Printf("Length: %s\n", r.Length)
	fmt.Printf("Balance: %s\n", r.Balance)
	fmt.Printf("Swing Weight: %s\n", r.SwingWeight)
	fmt.Printf("Beam Width: %s\n", r.BeamWidth)
	fmt.Printf("Tip/Shaft: %s\n", r.TipOrShaft)
	fmt.Printf("Composition: %s\n", r.Composition)
	fmt.Printf("Power Level: %s\n", r.PowerLevel)
	fmt.Printf("Stiffness: %s\n", r.Stiffness)
	fmt.Printf("String Pattern: %s\n", r.StringPattern)
	fmt.Printf("Main Skip: %s\n", r.MainSkip)
	fmt.Printf("String Tension: %s\n", r.StringTension)
}

// fmt.Printf is a helper function to print a field with a label
// func fmt.Printf(label string, value *string) {
// 	if value != nil {
// 		fmt.Printf("%s: %s\n", label, *value) // Dereference safely
// 	} else {
// 		fmt.Printf("%s: not set\n", label) // Handle the nil case
// 	}
// }

func New(ctx context.Context, cfg *config.Config) *Scraper {
	return &Scraper{
		ctx:           ctx,
		actionTimeout: cfg.ActionTimeout,
		debug:         cfg.Debug,
	}
}

// runWithTimeout runs an action with a specific timeout
func (s *Scraper) runWithTimeout(actions ...chromedp.Action) error {
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("parent context canceled: %w", s.ctx.Err())
	default:
		timeoutCtx, cancel := context.WithTimeout(s.ctx, s.actionTimeout)
		defer cancel()

		err := chromedp.Run(timeoutCtx, actions...)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return fmt.Errorf("action context canceled during execution")
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("action timed out after %v", s.actionTimeout)
			}
			return err
		}
		return nil
	}
}

// GetOptions handles initial setup and gets the list of options
func (s *Scraper) GetOptions(url string) ([]RacquetSpecs, error) {
	// var results []Result

	// Initial context check
	if err := s.ctx.Err(); err != nil {
		return nil, fmt.Errorf("initial context error: %w", err)
	}

	// Navigate and setup
	if err := s.setupPage(url); err != nil {
		return nil, err
	}

	// Get options
	var options []string
	if err := s.runWithTimeout(
		chromedp.Click(`.drop_arrow`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.optionslist li'))
				.map(el => el.textContent.trim())
				.filter(text => text !== '' && text !== 'Select')
		`, &options),
	); err != nil {
		return nil, fmt.Errorf("options error: %w", err)
	}

	// Log the filtered options
	log.Printf("Found %d valid options", len(options))
	for i, opt := range options {
		log.Printf("Option %d: %s", i+1, opt)
	}

	// Process each option
	return s.processOptions(options)
}

// setupPage handles navigation and initial page setup
func (s *Scraper) setupPage(url string) error {
	log.Println("Starting navigation...")

	// Navigation with context verification
	navCtx, navCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer navCancel()

	if err := chromedp.Run(navCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Checking browser status...")
			return nil
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	log.Println("Navigation complete, waiting for page load...")

	// Verify page load with separate context
	loadCtx, loadCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer loadCancel()

	if err := chromedp.Run(loadCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Verifying page load...")
			return nil
		}),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("page load verification failed: %w", err)
	}

	// Uncheck the checkbox
	log.Println("Unchecking current checkbox...")
	if err := s.runWithTimeout(
		chromedp.WaitVisible(`#currentcheckbox`, chromedp.ByID),
		chromedp.Evaluate(`
			const checkbox = document.getElementById('currentcheckbox');
			if (checkbox.checked) {
				checkbox.click();
			}
		`, nil),
		chromedp.Sleep(1*time.Second),
	); err != nil {
		return fmt.Errorf("failed to uncheck checkbox: %w", err)
	}

	return nil
}

// processOptions handles processing each individual option
func (s *Scraper) processOptions(options []string) ([]RacquetSpecs, error) {
	var results []RacquetSpecs

	for i, option := range options {
		specs := []RacquetSpecs{}
		log.Printf("Processing option %d/%d: %s", i+1, len(options), option)

		// result := Result{Option: option}

		// Click option and search
		if err := s.runWithTimeout(
			chromedp.Click(fmt.Sprintf(`//li[contains(text(), "%s")]`, option), chromedp.BySearch),
			chromedp.Sleep(1*time.Second),
			chromedp.Click(`#search_button`, chromedp.ByID),
			chromedp.Sleep(2*time.Second),
		); err != nil {
			log.Printf("Error processing option %s: %v", option, err)
			continue
		}

		// Get results
		var racNames []string
		if err := s.runWithTimeout(
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.rac_info .rac_name'))
					.map(el => el.textContent.trim())
			`, &racNames),
		); err != nil {
			log.Printf("Error getting results for option %s: %v", option, err)
			continue
		}

		// Print RAC names as we find them
		if len(racNames) > 0 {
			log.Printf("Found %d RAC names for option '%s':", len(racNames), option)
			for j, name := range racNames {
				log.Printf("%d. %s", j+1, name)
			}
			specs, err := s.getRacquetInfo(racNames)
			for _, spec := range specs {
				spec.Print()
			}
			if err != nil {
				log.Printf("Error getting specs for racquets: %v", err)
				continue
			}
		} else {
			log.Printf("No RAC names found for option '%s'", option)
		}

		// result.RacNames = racNames
		results = append(results, specs...)

		// Click dropdown for next option
		if err := s.runWithTimeout(
			chromedp.Click(`.drop_arrow`, chromedp.ByQuery),
			chromedp.Sleep(1*time.Second),
		); err != nil {
			log.Printf("Error reopening dropdown: %v", err)
			continue
		}
	}

	return results, nil
}

func (s *Scraper) getRacquetInfo(racNames []string) ([]RacquetSpecs, error) {
	var specs []RacquetSpecs

	for _, name := range racNames {
		// Find the racquet info div
		var racquetInfo RacquetSpecs
		var stringifySpecs string
		// racquetInfo.name = name

		// Get specs using JavaScript evaluation
		if err := s.runWithTimeout(
			chromedp.Evaluate(`
				(() => {
					// Find the div containing this racquet name
					// const racDiv = Array.from(document.querySelectorAll('#rac_name'))
					// 	.find(el => el.textContent.trim() === `+"`"+name+"`"+`);
					let foundDiv = null;
					let racNameDivs = document.querySelectorAll('.rac_name');
					for (let i = 0; i < racNameDivs.length; i++) {
						let currDiv = racNameDivs[i];
						if (currDiv.textContent.trim() === `+"`"+name+"`"+`) foundDiv = currDiv;
					}
					if (!foundDiv) return {
						name: `+"`"+name+"`"+`,
						error: "Couldn't find racDiv"
					};
					let parent = foundDiv.parentNode;
					console.dir(parent);
					if (!parent) {
						return {
							name: `+"`"+name+"`"+`,
							error: "Couldn't find parent"
						};
					}

					// Helper to find spec value
					const getSpec = (label) => {
						console.log("label: ", label);
						let trList = parent.querySelectorAll('tr');
						for (let i = 0; i < trList.length; i++) {
							let currTr = trList[i];
							if (currTr.textContent.trim().startsWith(label)) {
								console.log(currTr.querySelector('td').textContent.trim());
								return currTr.querySelector('td').textContent.trim();
							}
						}
						return "";
					};

					let headSize = getSpec('Head Size:');
					console.log("headSize: ", headSize);

					return JSON.stringify({
						name: `+"`"+name+"`"+`,
						error: '',
						headSize: headSize,
						length: getSpec('Length:'),
						balance: getSpec('Balance:'),
						swingWeight: getSpec('Swing Weight:'),
						beamWidth: getSpec('Beam Width:'),
						tipOrShaft: getSpec('Tip/Shaft:'),
						composition: getSpec('Composition:'),
						powerLevel: getSpec('Power Level:'),
						stiffness: getSpec('Stiffness:'),
						stringPattern: getSpec('String Pattern:'),
						mainSkip: getSpec('Main Skip:'),
						stringTension: getSpec('String Tension:')
					});
				})()
			`, &stringifySpecs),
		); err != nil {
			log.Printf("Error getting specs for racquet %s: %v", name, err)
			continue
		}

		// Unmarshal JSON string into racquetInfo struct
		if err := json.Unmarshal([]byte(stringifySpecs), &racquetInfo); err != nil {
			log.Printf("Error unmarshalling JSON for racquet %s: %v", name, err)
			continue
		}

		specs = append(specs, racquetInfo)
	}

	return specs, nil
}
