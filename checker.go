package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"rsc.io/pdf"
)

// Extract text from a PDF
func extractText(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	for i := 0; i < r.NumPage(); i++ {
		page := r.Page(i + 1)
		if page.V.IsNull() {
			continue
		}
		content := page.Content()
		for _, text := range content.Text {
			buf.WriteString(text.S)
			buf.WriteString(" ")
		}
	}
	// Normalize spaces and lowercase
	normalized := strings.ToLower(strings.Join(strings.Fields(buf.String()), " "))
	return normalized, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go file1.pdf file2.pdf")
		return
	}

	file1 := os.Args[1]
	file2 := os.Args[2]

	text1, err := extractText(file1)
	if err != nil {
		log.Fatalf("Error extracting %s: %v", file1, err)
	}
	text2, err := extractText(file2)
	if err != nil {
		log.Fatalf("Error extracting %s: %v", file2, err)
	}

	if text1 == text2 {
		fmt.Println("✅ PDFs are textually identical (fully recognizable content).")
	} else {
		// Save outputs for manual diffing
		_ = os.WriteFile("pdf1.txt", []byte(text1), 0644)
		_ = os.WriteFile("pdf2.txt", []byte(text2), 0644)
		fmt.Println("❌ PDFs differ in recognizable content. Check pdf1.txt and pdf2.txt for details.")
	}
}
