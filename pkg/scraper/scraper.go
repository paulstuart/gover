package scraper

import (
	"cmp"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/paulstuart/gollm/gover/pkg/model"
)

// ScrapeReleaseHistory scrapes https://go.dev/doc/devel/release to get all major Go versions and their release dates.
func ScrapeReleaseHistory() (map[string]string, error) {
	releaseDates := make(map[string]string)
	var mu sync.Mutex

	c := colly.NewCollector(
		colly.AllowedDomains("go.dev"),
	)
	c.UserAgent = "gollm-gover-scraper/1.0 (+https://github.com/paulstuart/gollm/gover)"

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Release history request URL: %s failed with response: %d, error: %v", r.Request.URL, r.StatusCode, err)
	})

	// Current format: <h2>go1.24.0 (released 2025-02-11)</h2>
	// We extract version and date from the heading text directly.
	versionDateRe := regexp.MustCompile(`(go1\.\d+)(?:\.\d+)?\s+\(released\s+(\d{4}-\d{2}-\d{2})\)`)

	c.OnHTML("h2", func(e *colly.HTMLElement) {
		text := e.Text
		matches := versionDateRe.FindStringSubmatch(text)
		if len(matches) >= 3 {
			version := matches[1]    // e.g., "go1.24"
			releaseDate := matches[2] // e.g., "2025-02-11"

			mu.Lock()
			// Only store first occurrence (the .0 release) for each major version
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

// ScrapeGoVersions scrapes the go.dev documentation for specified Go versions.
// It now accepts a map of versions to their release dates.
func ScrapeGoVersions(versions []string, versionReleaseDates map[string]string) ([]model.VersionData, error) {
	var allVersionData []model.VersionData
	var mu sync.Mutex // Mutex to protect allVersionData slice
	var wg sync.WaitGroup

	c := colly.NewCollector(
		colly.AllowedDomains("go.dev"),
		colly.Async(true), // Enable asynchronous requests
	)

	c.UserAgent = "gollm-gover-scraper/1.0 (+https://github.com/paulstuart/gollm/gover)"

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

		versionData := model.VersionData{
			Version: version,
			Changes: []model.ChangeCategory{},
		}

		// Populate ReleaseDate from the map
		if date, ok := versionReleaseDates[version]; ok {
			versionData.ReleaseDate = date
		} else {
			log.Printf("Warning: Release date not found for %s", version)
		}

		mainTitle := e.ChildText("h1")
		if mainTitle != "" {
			log.Printf("Main Title for %s: %s", version, mainTitle)
			versionData.Changes = append(versionData.Changes, model.ChangeCategory{
				Category:    "Overview",
				Description: mainTitle,
			})
		}

		e.ForEach("h2", func(_ int, el *colly.HTMLElement) {
			categoryName := el.Text
			log.Printf("  Found category: %s", categoryName)

			currentCategory := model.ChangeCategory{
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

	// Sort by version number descending (e.g., go1.24 before go1.23)
	slices.SortFunc(allVersionData, func(a, b model.VersionData) int {
		return -cmp.Compare(parseVersionMinor(a.Version), parseVersionMinor(b.Version))
	})

	return allVersionData, nil
}

// parseVersionMinor extracts the minor version number from a version string like "go1.24".
// Returns 0 if parsing fails.
func parseVersionMinor(version string) int {
	// Expected format: "go1.X" where X is the minor version
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
	// A simple way for now, assuming URL format go.dev/doc/go1.X
	// This can be made more robust if needed.
	if len(url) < 4 || url[len(url)-1] == '/' { // Basic check for short or trailing slash URLs
		return ""
	}

	// Find the last slash
	lastSlash := 0
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	// Extract the segment after the last slash
	if lastSlash != 0 && lastSlash < len(url)-1 {
		segment := url[lastSlash+1:]
		if len(segment) >= 3 && segment[:2] == "go" { // Check if it starts with "go"
			return segment
		}
	}
	return ""
}
