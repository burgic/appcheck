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
	"strings"
	"time"
	"context"

	"github.com/chromedp/chromedp"
	"github.com/PuerkitoBio/goquery"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type AppInfo struct {
	Timestamp    string
	CoinbaseRank string
	OKXRank      string
	TrustRank    string
}

func main() {
	countries := []string{
		"united-states",  // USA
		"united-kingdom", // UK
		// Add more countries if needed
	}

	for _, country := range countries {
		appData := scrapeTopApps(country)
		filename := saveToCSV(appData, country)
		fmt.Printf("Scraped data for %s and saved to %s\n", country, filename)
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

func randomDelay() {
	delay := rng.Intn(2) + 2
	time.Sleep(time.Duration(delay) * time.Second)
}

func scrapeTopApps(country string) AppInfo {
	baseURL := fmt.Sprintf("https://appfigures.com/top-apps/ios-app-store/%s/iphone/finance?list=free", country)
	var appData AppInfo
	appData.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	fmt.Printf("Scraping top apps for country: %s\n", country)

	// Create a new chrome instance
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Create a timeout
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Navigate to the page and wait for it to load
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible("div.s445742525-0", chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("Error navigating to page for %s: %v", country, err)
		return appData
	}

	// Scroll the page
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil),
		chromedp.Sleep(2*time.Second), // Wait for content to load
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

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Error parsing HTML for %s: %v", country, err)
		return appData
	}

	// Find and assign ranks for specified apps
	doc.Find("a.s-4262409-0").Each(func(i int, s *goquery.Selection) {
		rankAndNameText := strings.TrimSpace(s.Text())
		rankAndNameParts := strings.SplitN(rankAndNameText, ".", 2)
		if len(rankAndNameParts) != 2 {
			return
		}

		rank := strings.TrimSpace(rankAndNameParts[0])
		name := strings.TrimSpace(rankAndNameParts[1])

		// Assign rank to the respective app based on name
		switch name {
		case "Coinbase: Buy Bitcoin & Ether":
			appData.CoinbaseRank = rank
		case "OKX: Buy Bitcoin BTC & Crypto":
			appData.OKXRank = rank
		case "Trust: Crypto & Bitcoin Wallet":
			appData.TrustRank = rank
		}
	})

	return appData
}

func saveToCSV(appData AppInfo, country string) string {
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

	// Write headers only if the file is new
	headers := []string{"Timestamp", "CoinbaseRank", "OKXRank", "TrustRank"}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Error writing CSV headers: %v", err)
	}

	// Write the record with ranks and timestamp
	record := []string{
		appData.Timestamp,
		appData.CoinbaseRank,
		appData.OKXRank,
		appData.TrustRank,
	}
	if err := writer.Write(record); err != nil {
		log.Printf("Error writing app record: %v", err)
	}

	return filename
}
