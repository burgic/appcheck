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
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type AppInfo struct {
	Rank      string
	Name      string
	Pricing   string
	Developer string
}

func main() {
	apps := scrapeTopApps()
	filename := saveToCSV(apps)
	fmt.Printf("Scraped %d apps and saved to %s\n", len(apps), filename)
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

func scrapeTopApps() []AppInfo {
    baseURL := "https://appfigures.com/top-apps/ios-app-store/united-states/iphone/finance?list=free"
    var apps []AppInfo

    fmt.Printf("Scraping top apps: %s\n", baseURL)

    doc, err := fetchPage(baseURL)
    if err != nil {
        log.Printf("Error fetching page: %v", err)
        return nil
    }

    appBlocks := doc.Find("div.s445742525-0")
    if appBlocks.Length() == 0 {
        log.Printf("No app blocks found.")
        return nil
    }

    appBlocks.Each(func(i int, s *goquery.Selection) {
        app, valid := scrapeAppBlock(s)

        // If the app data is not valid, skip to the next iteration
        if !valid {
            return
        }

        // Convert the rank to an integer and stop if the rank is greater than 100
        rank, err := strconv.Atoi(app.Rank)
        if err != nil {
            log.Printf("Error converting rank: %v", err)
            return
        }

        if rank > 70 {
            log.Printf("Stopping scrape after rank 70")
            return // Stop processing further apps
        }

        apps = append(apps, app)
        fmt.Printf("Scraped: %s (Rank: %s)\n", app.Name, app.Rank)
        randomDelay()
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



func fetchPage(url string) (*goquery.Document, error) {
	resp, err := makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func saveToCSV(apps []AppInfo) string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join("results", fmt.Sprintf("apps_%s.csv", timestamp))

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
			
		}
		if err := writer.Write(record); err != nil {
			log.Printf("Error writing app record: %v", err)
		}
	}

	return filename
}
