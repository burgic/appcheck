package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"context"

	"github.com/chromedp/chromedp"
	"github.com/PuerkitoBio/goquery"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type AppInfo struct {
	Timestamp       string
	US_CoinbaseRank string
	US_OKXRank      string
	US_TrustRank    string
	UK_CoinbaseRank string
	UK_OKXRank      string
	UK_TrustRank    string
}

func main() {
	appData := AppInfo{Timestamp: time.Now().Format("2006-01-02 15:04:05")}

	// Scrape data for each country and update the `appData` struct
	appData = scrapeTopApps("united-states", appData, "US")
	appData = scrapeTopApps("united-kingdom", appData, "UK")

	// Save the combined results into a single CSV file
	filename := saveToCSV(appData)
	fmt.Printf("Scraped data saved to %s\n", filename)
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

func randomDelay() {
	delay := rng.Intn(2) + 2
	time.Sleep(time.Duration(delay) * time.Second)
}

func scrapeTopApps(country string, appData AppInfo, prefix string) AppInfo {
	baseURL := fmt.Sprintf("https://appfigures.com/top-apps/ios-app-store/%s/iphone/finance?list=free", country)

	fmt.Printf("\nScraping top apps for country: %s\n", country)

	// Create a new Chrome instance
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Create a timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)  // Increased timeout
	defer cancel()

	// Navigate to the page and wait for it to load
	log.Printf("Navigating to %s", baseURL)
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible("div.s-1362551351-0", chromedp.ByQuery),
		// Add a longer wait to ensure content loads
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		log.Printf("Error navigating to page for %s: %v", country, err)
		return appData
	}

	// Scroll multiple times to ensure all content loads
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			function scrollDown() {
				window.scrollTo(0, document.body.scrollHeight/2);
				setTimeout(() => {
					window.scrollTo(0, document.body.scrollHeight);
				}, 1000);
			}
			scrollDown();
		`, nil),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		log.Printf("Error scrolling page for %s: %v", country, err)
		return appData
	}

	// Extract the HTML content
	var html string
	err = chromedp.Run(ctx,
		chromedp.OuterHTML("body", &html),
	)
	if err != nil {
		log.Printf("Error extracting HTML for %s: %v", country, err)
		return appData
	}

	// Debug: Log HTML length
	log.Printf("Retrieved HTML length: %d characters", len(html))

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Error parsing HTML for %s: %v", country, err)
		return appData
	}

	// Debug: Print all found apps
	log.Printf("\nAll apps found for %s:", country)
	count := 0
	doc.Find("div.s-1362551351-0").Each(func(i int, s *goquery.Selection) {
		appLink := s.Find("a.s-4262409-0")
		text := strings.TrimSpace(appLink.Text())
		title := appLink.AttrOr("title", "no title")
		log.Printf("Found app %d: Text='%s', Title='%s'", i+1, text, title)
		count++
	})
	log.Printf("Total apps found: %d\n", count)

	// Now do the actual scraping
	doc.Find("div.s-1362551351-0").Each(func(i int, s *goquery.Selection) {
		appLink := s.Find("a.s-4262409-0")
		rankAndNameText := strings.TrimSpace(appLink.Text())
		title := appLink.AttrOr("title", "")
		
		// Debug: Print each app being processed
		log.Printf("Processing: '%s' (title: '%s')", rankAndNameText, title)

		rankAndNameParts := strings.SplitN(rankAndNameText, ".", 2)
		if len(rankAndNameParts) != 2 {
			log.Printf("Skipping invalid format: %s", rankAndNameText)
			return
		}

		rank := strings.TrimSpace(rankAndNameParts[0])
		name := strings.TrimSpace(rankAndNameParts[1])

		// Debug: Print when we find a potential match
		if strings.Contains(name, "Coinbase") || strings.Contains(title, "Coinbase") {
			log.Printf("Found Coinbase: %s", rankAndNameText)
		}
		if strings.Contains(name, "OKX") || strings.Contains(title, "OKX") {
			log.Printf("Found OKX: %s", rankAndNameText)
		}
		if strings.Contains(name, "Trust") || strings.Contains(title, "Trust") {
			log.Printf("Found Trust: %s", rankAndNameText)
		}

		// Assign ranks
		switch {
		case strings.Contains(name, "Coinbase") || strings.Contains(title, "Coinbase"):
			if prefix == "US" {
				appData.US_CoinbaseRank = rank
			} else if prefix == "UK" {
				appData.UK_CoinbaseRank = rank
			}
		case strings.Contains(name, "OKX") || strings.Contains(title, "OKX"):
			if prefix == "US" {
				appData.US_OKXRank = rank
			} else if prefix == "UK" {
				appData.UK_OKXRank = rank
			}
		case strings.Contains(name, "Trust") || strings.Contains(title, "Trust"):
			if prefix == "US" {
				appData.US_TrustRank = rank
			} else if prefix == "UK" {
				appData.UK_TrustRank = rank
			}
		}
	})

	// Debug: Print final results for this country
	log.Printf("\nFinal results for %s:", country)
	if prefix == "US" {
		log.Printf("Coinbase: %s, OKX: %s, Trust: %s", 
			appData.US_CoinbaseRank, 
			appData.US_OKXRank, 
			appData.US_TrustRank)
	} else {
		log.Printf("Coinbase: %s, OKX: %s, Trust: %s", 
			appData.UK_CoinbaseRank, 
			appData.UK_OKXRank, 
			appData.UK_TrustRank)
	}

	return appData
}

func saveToCSV(appData AppInfo) string {
	filename := "results/apps_ranks.csv"

	// Ensure the results directory exists
	if err := os.MkdirAll("results", os.ModePerm); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}

	// Open the CSV file in append mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Check if we need to write the header (file is new or empty)
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	if fileInfo.Size() == 0 {
		headers := []string{"Timestamp", "US_CoinbaseRank", "US_OKXRank", "US_TrustRank", "UK_CoinbaseRank", "UK_OKXRank", "UK_TrustRank"}
		if err := writer.Write(headers); err != nil {
			log.Fatalf("Error writing CSV headers: %v", err)
		}
	}

	// Write the record with ranks and timestamp for both countries
	record := []string{
		appData.Timestamp,
		appData.US_CoinbaseRank,
		appData.US_OKXRank,
		appData.US_TrustRank,
		appData.UK_CoinbaseRank,
		appData.UK_OKXRank,
		appData.UK_TrustRank,
	}
	if err := writer.Write(record); err != nil {
		log.Printf("Error writing app record: %v", err)
	}

	return filename
}