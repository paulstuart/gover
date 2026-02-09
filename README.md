# gover - Go Version Scraper

`gover` is a Go application designed to scrape release notes and documentation for various Go versions from `go.dev/doc/go{VERSION}`. The extracted data is then outputted as a JSON file, intended for use in training Large Language Models (LLMs) or for general machine-readable metadata about Go language changes.

## Goal

To generate clean, structured data about Go language evolution, package changes, and new features, providing accurate and expert knowledge for Golang-specific LLMs.

## Features

*   Scrapes `go.dev/doc/go{VERSION}` pages.
*   Extracts version overview and categorized changes.
*   Outputs data in a structured JSON format.
*   Uses `go-colly` for web scraping.

## Getting Started

### Build

To build the `gover` executable, run the following command in the project root:

```bash
go build -o gover cmd/gover/main.go
```

This will create an executable named `gover` in the current directory.

### Run

To run the scraper and generate a JSON output file:

```bash
./gover -versions go1.25,go1.26 -output go_version_data.json
```

**Flags:**

*   `-versions`: A comma-separated list of Go versions to scrape (e.g., `go1.25,go1.26`). Defaults to `go1.25,go1.26`.
*   `-output`: The path to the output JSON file. Defaults to `go_version_data.json`.

### Example Output

A sample of the expected JSON output structure can be found in `AGENTS.md`. The output will contain an array of `VersionData` objects, each detailing changes for a specific Go version.

## Project Structure

*   `cmd/gover`: Contains the `main.go` file, the entry point for the application.
*   `pkg/model`: Defines the Go structs used to model the scraped data for JSON output.
*   `pkg/scraper`: Contains the core web scraping logic using the `go-colly` library.
*   `internal/config`: (Planned for future use) For application configuration.

## Next Steps / Enhancements

*   Refine HTML parsing to extract more granular and hierarchical data.
*   Implement dynamic discovery of Go versions rather than hardcoding.
*   Improve error handling and logging.
*   Add comprehensive unit and integration tests.
*   Enhance example code extraction.
*   Scrape release dates from a reliable source.
