package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/paulstuart/gover"
)

func main() {
	outputFile := flag.String("output", "go_version_data.json", "Output JSON file path")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	versionData, err := gover.Scrape()
	if err != nil {
		log.Fatalf("Error scraping: %v", err)
	}

	jsonData, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(*outputFile, jsonData, 0644); err != nil {
		log.Fatalf("Error writing JSON to file %s: %v", *outputFile, err)
	}

	log.Printf("Successfully wrote scraped data to %s", *outputFile)
}
