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
./gover -versions -output go_version_data.json
```

**Flags:**

*   `-output`: The path to the output JSON file. Defaults to `go_version_data.json`.

### Data Structure

The resulting json file effectively mirrors the hierachical layout of the html for each major release note at https://go.dev/doc/devel/release, so it comprises a list of released versions (descending from latest release), with the release version and date and then the various aspects of Go that have been changed, e.g., tooling, packages, functions, etc.

## Next Steps / Enhancements

*   Refine HTML parsing to extract more granular and hierarchical data.
*   Improve error handling and logging.
*   Add comprehensive unit and integration tests.
