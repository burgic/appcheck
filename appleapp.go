package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type AppInfo struct {
	Name        string
	FinanceRank string
	Timestamp   string
}

func main() {
	// URL of the app page
	url := "https://apps.apple.com/us/app/coinbase-buy-bitcoin-ether/id886427730"
	
	// Scrape the app information
	app := scrapeAppPage(url)
	
	// Save the result to the CSV file, appending the data
	saveToCSV(app)
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

func fetchPage(url string) (*goquery.Document, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func scrapeAppPage(url string) AppInfo {
	doc, err := fetchPage(url)
	if err != nil {
		log.Fatalf("Error fetching page: %v", err)
	}

	app := AppInfo{}

	// Scrape the app name
	nameElem := doc.Find("h1.product-header__title")
	app.Name = strings.TrimSpace(nameElem.Contents().FilterFunction(func(i int, s *goquery.Selection) bool {
		return goquery.NodeName(s) == "#text"
	}).Text())

	// Scrape the rank in Finance category
	rankElem := doc.Find("a.inline-list__item")
	app.FinanceRank = strings.TrimSpace(rankElem.Text())

	// Add current timestamp
	app.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	return app
}

func saveToCSV(app AppInfo) {
	// Define the folder and file paths
    folderPath := "./results"
    filename := fmt.Sprintf("%s/appleappcoinbase.csv", folderPath)

    // Check if the results directory exists, if not, create it
    if _, err := os.Stat(folderPath); os.IsNotExist(err) {
        fmt.Printf("Directory %s does not exist, creating...\n", folderPath)
        err := os.MkdirAll(folderPath, os.ModePerm)
        if err != nil {
            log.Fatalf("Failed to create directory: %v", err)
        }
    }

    // Check if file exists
    fileExists := false
    if _, err := os.Stat(filename); err == nil {
        fileExists = true
    }
	// Open the file in append mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header only if the file is new
	if !fileExists {
		headers := []string{"Name", "Rank", "Timestamp"}
		if err := writer.Write(headers); err != nil {
			log.Fatalf("Error writing CSV headers: %v", err)
		}
	}

	// Append the app data
	record := []string{app.Name, app.FinanceRank, app.Timestamp}
	if err := writer.Write(record); err != nil {
		log.Fatalf("Error writing record to CSV: %v", err)
	}

	fmt.Printf("Appended to %s: %s, %s, %s\n", filename, app.Name, app.FinanceRank, app.Timestamp)
}
