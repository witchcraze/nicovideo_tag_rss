package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <tag>", os.Args[0])
	}
	tag := os.Args[1]

	url := fmt.Sprintf("https://www.nicovideo.jp/tag/%s?sort=registeredAt&order=desc", tag)
	log.Printf("Fetching URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body: %v", err)
	}

	outDir := filepath.Join("nico", "testdata")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create dir: %v", err)
	}

	filename := filepath.Join(outDir, fmt.Sprintf("raw_%s.html", tag))
	if err := os.WriteFile(filename, body, 0644); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	log.Printf("Successfully saved HTML to %s", filename)
}
