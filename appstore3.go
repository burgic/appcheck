package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	// "strconv"
	"strings"
	"time"
	"context"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type AppInfo struct {
	Rank      string
	Name      string
	Pricing   string
	Developer string
}

func main() {
	countries := []string{
		"united-states",  // USA
		"united-kingdom", // UK
		// You can add more countries here, like "canada", "germany", etc.
	}

	for _, country := range countries {
		apps := scrapeTopAppsWithScroll(country)
		filename := saveToCSV(apps, country)
		fmt.Printf("Scraped %d apps for %s and saved to %s\n", len(apps), country, filename)
	}
}

func createClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return client
}

func makeRequest(url string) (*http.Response, error) {
	client := createClient()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func randomDelay() {
	delay := rng.Intn(2) + 2
	time.Sleep(time.Duration(delay) * time.Second)
}

// Scrape the apps with chromedp to handle infinite scroll
func scrapeTopAppsWithScroll(country string) []AppInfo {
    // Set a 2-minute timeout for chromedp context
    ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
    timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 2*time.Minute)
    defer cancel()
    defer timeoutCancel()

    // Define the base URL for the specific country
    baseURL := fmt.Sprintf("https://appfigures.com/top-apps/ios-app-store/%s/iphone/finance?list=free", country)

    // Variable to store the page's HTML content
    var pageHTML string

    // Navigate to the page and simulate scrolling
    err := chromedp.Run(timeoutCtx,
        chromedp.Navigate(baseURL),
        chromedp.WaitVisible(`div.s445742525-0`), // Wait for the first app block to be visible
        chromedp.ActionFunc(func(ctx context.Context) error {
            // Scroll the page to load more content
            for i := 0; i < 10; i++ { // Scroll multiple times to trigger loading more content
                err := chromedp.Run(ctx,
                    chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight);`, nil),
                    chromedp.Sleep(2*time.Second), // Wait for new content to load
                )
                if err != nil {
                    return err
                }
            }
            return nil
        }),
        chromedp.OuterHTML("html", &pageHTML), // Get the entire page's HTML after scrolling
    )

    if err != nil {
        log.Fatalf("Failed to load the page for %s: %v", country, err)
    }

    // Now parse the HTML with goquery to extract app info
    return parseAppsFromHTML(pageHTML)
}

// Parse apps from the HTML content
func parseAppsFromHTML(html string) []AppInfo {
    var apps []AppInfo
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        log.Fatalf("Error parsing HTML: %v", err)
    }

    // Find app blocks and scrape the relevant data
    doc.Find("div.s445742525-0").Each(func(i int, s *goquery.Selection) {
        app, valid := scrapeAppBlock(s)
        if valid {
            apps = append(apps, app)
        }
    })

    return apps
}

func scrapeAppBlock(s *goquery.Selection) (AppInfo, bool) {
	app := AppInfo{}

	// Extract rank and name from the <a> tag
	rankAndNameElem := s.Find("a.s-4262409-0")
	rankAndNameText := strings.TrimSpace(rankAndNameElem.Text())

	// Check if the rank and name text is valid
	if rankAndNameText == "" {
		log.Printf("Empty rank and name found, skipping entry.")
		return app, false // Invalid entry, return and indicate to skip
	}

	// Split the text by the period (.) to get rank and name
	rankAndNameParts := strings.SplitN(rankAndNameText, ".", 2) // Using SplitN to limit to two parts

	if len(rankAndNameParts) != 2 {
		log.Printf("Unexpected format for rank and name: %s", rankAndNameText)
		return app, false // Invalid entry, return and indicate to skip
	}

	app.Rank = strings.TrimSpace(rankAndNameParts[0]) // Rank is before the period
	app.Name = strings.TrimSpace(rankAndNameParts[1]) // Name is after the period

	return app, true
}

func saveToCSV(apps []AppInfo, country string) string {
    timestamp := time.Now().Format("2006-01-02_15-04-05")
    filename := filepath.Join("results", fmt.Sprintf("apps_%s_%s.csv", country, timestamp))

    // Ensure the results directory exists
    if err := os.MkdirAll("results", os.ModePerm); err != nil {
        log.Fatalf("Failed to create results directory: %v", err)
    }

    file, err := os.Create(filename)
    if err != nil {
        log.Fatalf("Failed to create file: %v", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    headers := []string{"Rank", "Name", "Pricing", "Developer"}
    if err := writer.Write(headers); err != nil {
        log.Fatalf("Error writing CSV headers: %v", err)
    }

    for _, app := range apps {
        record := []string{
            app.Rank,
            app.Name,
            app.Pricing,
            app.Developer,
        }
        if err := writer.Write(record); err != nil {
            log.Printf("Error writing app record: %v", err)
        }
    }

    return filename
}