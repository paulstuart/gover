// Package gover provides Go version information by scraping go.dev documentation.
package gover

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// VersionData represents the data collected for a specific Go version.
type VersionData struct {
	Version     string           `json:"version"`
	ReleaseDate string           `json:"releaseDate,omitempty"`
	Changes     []ChangeCategory `json:"changes"`
}

// ChangeCategory represents a high-level category of changes (e.g., "Language Changes", "Core Library").
type ChangeCategory struct {
	Category    string         `json:"category"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Examples    []string       `json:"examples,omitempty"`
	Package     string         `json:"package,omitempty"`
	Changes     []SymbolChange `json:"changes,omitempty"`
}

// SymbolChange represents a specific change to a function, method, or type within a package.
type SymbolChange struct {
	Type        string `json:"type"`        // e.g., "added", "changed", "obsoleted"
	Symbol      string `json:"symbol"`      // e.g., "http.NewRequestWithContext"
	Description string `json:"description"` // Description of the specific change
}

const goVersionsURL = "https://go.dev/VERSION?m=text"

// Scrape fetches Go version information from go.dev and returns a slice of VersionData.
func Scrape() ([]VersionData, error) {
	latestVersion, err := getLatestGoVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest Go version: %w", err)
	}
	log.Printf("Latest Go version: %s", latestVersion)

	majorVersion, err := extractMajorVersion(latestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to extract major version: %w", err)
	}
	log.Printf("Latest major version: %d", majorVersion)

	versions := generateVersionStrings(majorVersion)
	log.Printf("Will scrape versions: %v", versions)

	log.Println("Scraping release history for dates...")
	releaseDates, err := scrapeReleaseHistory()
	if err != nil {
		return nil, fmt.Errorf("error scraping release history: %w", err)
	}
	log.Printf("Found release dates for %d versions", len(releaseDates))

	log.Printf("Starting scraping for version details...")
	versionData, err := scrapeGoVersions(versions, releaseDates)
	if err != nil {
		return nil, fmt.Errorf("error during scraping: %w", err)
	}
	log.Printf("Finished scraping. Found data for %d versions.", len(versionData))

	return versionData, nil
}

// getLatestGoVersion fetches the current Go version string from go.dev.
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

// scrapeReleaseHistory scrapes https://go.dev/doc/devel/release to get all major Go versions and their release dates.
func scrapeReleaseHistory() (map[string]string, error) {
	releaseDates := make(map[string]string)
	var mu sync.Mutex

	c := colly.NewCollector(
		colly.AllowedDomains("go.dev"),
	)
	c.UserAgent = "gover-scraper/1.0 (+https://github.com/paulstuart/gover)"

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Release history request URL: %s failed with response: %d, error: %v", r.Request.URL, r.StatusCode, err)
	})

	versionDateRe := regexp.MustCompile(`(go1\.\d+)(?:\.\d+)?\s+\(released\s+(\d{4}-\d{2}-\d{2})\)`)

	c.OnHTML("h2", func(e *colly.HTMLElement) {
		text := e.Text
		matches := versionDateRe.FindStringSubmatch(text)
		if len(matches) >= 3 {
			version := matches[1]
			releaseDate := matches[2]

			mu.Lock()
			if _, exists := releaseDates[version]; !exists {
				releaseDates[version] = releaseDate
				log.Printf("Found release: %s, Date: %s", version, releaseDate)
			}
			mu.Unlock()
		}
	})

	err := c.Visit("https://go.dev/doc/devel/release")
	if err != nil {
		return nil, fmt.Errorf("failed to visit release history page: %w", err)
	}
	c.Wait()

	if len(releaseDates) == 0 {
		return nil, fmt.Errorf("no release dates found on https://go.dev/doc/devel/release")
	}

	return releaseDates, nil
}

// scrapeGoVersions scrapes the go.dev documentation for specified Go versions.
func scrapeGoVersions(versions []string, versionReleaseDates map[string]string) ([]VersionData, error) {
	var allVersionData []VersionData
	var mu sync.Mutex
	var wg sync.WaitGroup

	c := colly.NewCollector(
		colly.AllowedDomains("go.dev"),
		colly.Async(true),
	)

	c.UserAgent = "gover-scraper/1.0 (+https://github.com/paulstuart/gover)"

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response: %d, error: %v", r.Request.URL, r.StatusCode, err)
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		version := extractVersionFromURL(e.Request.URL.String())
		if version == "" {
			log.Printf("Could not extract version from URL: %s", e.Request.URL.String())
			return
		}

		log.Printf("Processing content for Go version: %s", version)

		versionData := VersionData{
			Version: version,
			Changes: []ChangeCategory{},
		}

		if date, ok := versionReleaseDates[version]; ok {
			versionData.ReleaseDate = date
		} else {
			log.Printf("Warning: Release date not found for %s", version)
		}

		mainTitle := e.ChildText("h1")
		if mainTitle != "" {
			log.Printf("Main Title for %s: %s", version, mainTitle)
			versionData.Changes = append(versionData.Changes, ChangeCategory{
				Category:    "Overview",
				Description: mainTitle,
			})
		}

		e.ForEach("h2", func(_ int, el *colly.HTMLElement) {
			categoryName := el.Text
			log.Printf("  Found category: %s", categoryName)

			currentCategory := ChangeCategory{
				Category: categoryName,
			}

			nextSibling := el.DOM.Next()
			if nextSibling.Length() > 0 && nextSibling.Is("p") {
				currentCategory.Description = nextSibling.Text()
			}

			versionData.Changes = append(versionData.Changes, currentCategory)
		})

		mu.Lock()
		allVersionData = append(allVersionData, versionData)
		mu.Unlock()
		wg.Done()
	})

	for _, v := range versions {
		wg.Add(1)
		url := fmt.Sprintf("https://go.dev/doc/%s", v)
		log.Printf("Visiting: %s", url)
		c.Visit(url)
	}

	wg.Wait()

	slices.SortFunc(allVersionData, func(a, b VersionData) int {
		return -cmp.Compare(parseVersionMinor(a.Version), parseVersionMinor(b.Version))
	})

	return allVersionData, nil
}

// parseVersionMinor extracts the minor version number from a version string like "go1.24".
func parseVersionMinor(version string) int {
	if !strings.HasPrefix(version, "go1.") {
		return 0
	}
	minorStr := strings.TrimPrefix(version, "go1.")
	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return 0
	}
	return minor
}

// extractVersionFromURL is a helper to get the "go1.X" part from the URL.
func extractVersionFromURL(url string) string {
	if len(url) < 4 || url[len(url)-1] == '/' {
		return ""
	}

	lastSlash := 0
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash != 0 && lastSlash < len(url)-1 {
		segment := url[lastSlash+1:]
		if len(segment) >= 3 && segment[:2] == "go" {
			return segment
		}
	}
	return ""
}
