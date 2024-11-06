package main

import (
	//"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	//"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"context"

	"github.com/chromedp/chromedp"
	"github.com/PuerkitoBio/goquery"
)

// Define column headers as constants
const (
	DateHeader = "Date"
	TimeHeader = "Time"

	// iOS headers
	USiOSHeader  = "United States - iOS App Store"
	UKiOSHeader  = "United Kingdom - iOS App Store"
	
	// Play Store headers
	USPlayHeader = "United States - Google Play Store"
	UKPlayHeader = "United Kingdom - Google Play Store"

	// App headers
	CoinbaseHeader = "Coinbase"
	OKXHeader      = "OKX"
	TrustHeader    = "Trust Wallet"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
// Regular expression to clean up the rank and name text
var cleanupRegex = regexp.MustCompile(`<!--.*?-->`)

type AppInfo struct {
	Timestamp               string
	Date                   string
	Time                   string
	US_iOS_CoinbaseRank    string
	US_iOS_OKXRank         string
	US_iOS_TrustRank       string
	UK_iOS_CoinbaseRank    string
	UK_iOS_OKXRank         string
	UK_iOS_TrustRank       string
	US_Play_CoinbaseRank   string
	US_Play_OKXRank        string
	US_Play_TrustRank      string
	UK_Play_CoinbaseRank   string
	UK_Play_OKXRank        string
	UK_Play_TrustRank      string
}

func main() {
	now := time.Now()
	appData := AppInfo{
		Timestamp: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
		Time:      now.Format("15:04:05"),
	}

	// Scrape iOS App Store
	appData = scrapeTopApps("united-states", "ios", appData, "US")
	appData = scrapeTopApps("united-kingdom", "ios", appData, "UK")

	// Scrape Play Store
	appData = scrapeTopApps("united-states", "play", appData, "US")
	appData = scrapeTopApps("united-kingdom", "play", appData, "UK")

	// Save the combined results into a single CSV file
	filename := saveToCSV(appData)
	fmt.Printf("Scraped data saved to %s\n", filename)
}

func cleanText(text string) string {
	// Remove HTML comments
	text = cleanupRegex.ReplaceAllString(text, "")
	// Remove extra spaces and trim
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func getStoreURL(country, store string) string {
	if store == "ios" {
		return fmt.Sprintf("https://appfigures.com/top-apps/ios-app-store/%s/iphone/finance?list=free", country)
	}
	return fmt.Sprintf("https://appfigures.com/top-apps/google-play/%s/finance", country)
}

func scrapeTopApps(country, store string, appData AppInfo, prefix string) AppInfo {
	baseURL := getStoreURL(country, store)

	fmt.Printf("\nScraping %s store for country: %s\n", store, country)
	log.Printf("URL: %s", baseURL)

	// Create a new Chrome instance
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Create a timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Navigate to the page and wait for it to load
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible("a.s-4262409-0", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		log.Printf("Error navigating to page: %v", err)
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
		log.Printf("Error scrolling page: %v", err)
		return appData
	}

	// Extract the HTML content
	var html string
	err = chromedp.Run(ctx,
		chromedp.OuterHTML("body", &html),
	)
	if err != nil {
		log.Printf("Error extracting HTML: %v", err)
		return appData
	}

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return appData
	}

	// Debug: Print all found apps
	log.Printf("\nAll apps found for %s %s:", country, store)
	count := 0
	doc.Find("a.s-4262409-0").Each(func(i int, s *goquery.Selection) {
		text := cleanText(s.Text())
		title := s.AttrOr("title", "no title")
		log.Printf("Found app %d: Text='%s', Title='%s'", i+1, text, title)
		count++
	})
	log.Printf("Total apps found: %d\n", count)

	// Now do the actual scraping
	doc.Find("a.s-4262409-0").Each(func(i int, s *goquery.Selection) {
		rankAndNameText := cleanText(s.Text())
		title := s.AttrOr("title", "")
		
		log.Printf("Processing: '%s' (title: '%s')", rankAndNameText, title)

		rankAndNameParts := strings.SplitN(rankAndNameText, ".", 2)
		if len(rankAndNameParts) != 2 {
			log.Printf("Skipping invalid format: %s", rankAndNameText)
			return
		}

		rank := strings.TrimSpace(rankAndNameParts[0])
		name := strings.TrimSpace(rankAndNameParts[1])

		// Debug: Print matches
		if strings.Contains(name, "Coinbase") || strings.Contains(title, "Coinbase") {
			log.Printf("Found Coinbase: %s", rankAndNameText)
		}
		if strings.Contains(name, "OKX") || strings.Contains(title, "OKX") {
			log.Printf("Found OKX: %s", rankAndNameText)
		}
		if strings.Contains(name, "Trust") || strings.Contains(title, "Trust") {
			log.Printf("Found Trust: %s", rankAndNameText)
		}

		// Assign ranks based on store and country
		switch {
		case strings.Contains(name, "Coinbase") || strings.Contains(title, "Coinbase"):
			if prefix == "US" {
				if store == "ios" {
					appData.US_iOS_CoinbaseRank = rank
				} else {
					appData.US_Play_CoinbaseRank = rank
				}
			} else {
				if store == "ios" {
					appData.UK_iOS_CoinbaseRank = rank
				} else {
					appData.UK_Play_CoinbaseRank = rank
				}
			}
		case strings.Contains(name, "OKX") || strings.Contains(title, "OKX"):
			if prefix == "US" {
				if store == "ios" {
					appData.US_iOS_OKXRank = rank
				} else {
					appData.US_Play_OKXRank = rank
				}
			} else {
				if store == "ios" {
					appData.UK_iOS_OKXRank = rank
				} else {
					appData.UK_Play_OKXRank = rank
				}
			}
		case strings.Contains(name, "Trust") || strings.Contains(title, "Trust"):
			if prefix == "US" {
				if store == "ios" {
					appData.US_iOS_TrustRank = rank
				} else {
					appData.US_Play_TrustRank = rank
				}
			} else {
				if store == "ios" {
					appData.UK_iOS_TrustRank = rank
				} else {
					appData.UK_Play_TrustRank = rank
				}
			}
		}
	})

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
		// Write blank line
		writer.Write([]string{})

		// Write timestamp headers
		timestampHeaders := []string{DateHeader, TimeHeader}
		emptySlice := make([]string, 12) // 12 empty columns for alignment
		writer.Write(append(timestampHeaders, emptySlice...))

		// Write store headers
		storeHeaders := []string{
			"", "", // Date & Time columns
			USiOSHeader, "", "", // 3 columns for US iOS
			UKiOSHeader, "", "", // 3 columns for UK iOS
			USPlayHeader, "", "", // 3 columns for US Play
			UKPlayHeader, "", "", // 3 columns for UK Play
		}
		writer.Write(storeHeaders)

		// Write app headers
		appHeaders := []string{
			"", "", // Date & Time columns
			CoinbaseHeader, OKXHeader, TrustHeader, // US iOS
			CoinbaseHeader, OKXHeader, TrustHeader, // UK iOS
			CoinbaseHeader, OKXHeader, TrustHeader, // US Play
			CoinbaseHeader, OKXHeader, TrustHeader, // UK Play
		}
		writer.Write(appHeaders)
	}

	// Write the data row
	record := []string{
		appData.Date,
		appData.Time,
		appData.US_iOS_CoinbaseRank,
		appData.US_iOS_OKXRank,
		appData.US_iOS_TrustRank,
		appData.UK_iOS_CoinbaseRank,
		appData.UK_iOS_OKXRank,
		appData.UK_iOS_TrustRank,
		appData.US_Play_CoinbaseRank,
		appData.US_Play_OKXRank,
		appData.US_Play_TrustRank,
		appData.UK_Play_CoinbaseRank,
		appData.UK_Play_OKXRank,
		appData.UK_Play_TrustRank,
	}
	if err := writer.Write(record); err != nil {
		log.Printf("Error writing app record: %v", err)
	}

	return filename
}