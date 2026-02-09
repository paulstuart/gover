package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings" // Added missing import

	"github.com/paulstuart/gover/pkg/scraper"
)

func main() {
	outputFile := flag.String("output", "go_version_data.json", "Output JSON file path")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	latestVersion, err := getLatestGoVersion()
	if err != nil {
		log.Fatalf("Failed to get latest Go version: %v", err)
	}
	log.Printf("Latest Go version: %s", latestVersion)

	majorVersion, err := extractMajorVersion(latestVersion)
	if err != nil {
		log.Fatalf("Failed to extract major version: %v", err)
	}
	log.Printf("Latest major version: %d", majorVersion)

	versions := generateVersionStrings(majorVersion)
	log.Printf("Will scrape versions: %v", versions)

	log.Println("Scraping release history for dates...")
	releaseDates, err := scraper.ScrapeReleaseHistory()
	if err != nil {
		log.Fatalf("Error scraping release history: %v", err)
	}
	log.Printf("Found release dates for %d versions", len(releaseDates))

	log.Printf("Starting scraping for version details...")
	versionData, err := scraper.ScrapeGoVersions(versions, releaseDates)
	if err != nil {
		log.Fatalf("Error during scraping: %v", err)
	}
	log.Printf("Finished scraping. Found data for %d versions.", len(versionData))

	jsonData, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	err = os.WriteFile(*outputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing JSON to file %s: %v", *outputFile, err)
	}

	log.Printf("Successfully wrote scraped data to %s", *outputFile)
}

const goVersionsURL = "https://go.dev/VERSION?m=text"

// getLatestGoVersion fetches the current Go version string from go.dev.
// Returns the version string like "go1.24.0".
func getLatestGoVersion() (string, error) {
	resp, err := http.Get(goVersionsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch Go versions, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	firstLine := strings.SplitN(string(body), "\n", 2)[0]
	return strings.TrimSpace(firstLine), nil
}

// extractMajorVersion parses a Go version string and returns the major version number.
// For example, "go1.24.0" returns 24.
func extractMajorVersion(versionString string) (int, error) {
	re := regexp.MustCompile(`go1\.(\d+)`)
	matches := re.FindStringSubmatch(versionString)

	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse major version from: %s", versionString)
	}

	majorVersion, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("could not convert major version to int: %w", err)
	}
	return majorVersion, nil
}

// generateVersionStrings creates a list of Go version strings from go1.1 to go1.<majorVersion>.
func generateVersionStrings(majorVersion int) []string {
	versions := make([]string, 0, majorVersion)
	for i := 1; i <= majorVersion; i++ {
		versions = append(versions, fmt.Sprintf("go1.%d", i))
	}
	return versions
}
