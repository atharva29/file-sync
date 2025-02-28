package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/xuri/excelize/v2"
)

// Define a struct to represent mark data record
type MarkRecord struct {
	Date string
	Time string
	Data string
}

func main() {
	// Define command-line flags
	inputFile := pflag.StringP("input", "i", "input.txt", "Path to input file containing mark data")
	outputFile := pflag.StringP("output", "o", "output.xlsx", "Path to output Excel file")
	pflag.Parse()

	// Validate input
	if *inputFile == "" || *outputFile == "" {
		fmt.Println("Error: Both input and output file paths are required")
		fmt.Println("Usage: ./mark-processor --input/-i input_file.txt --output/-o output_file.xlsx")
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: Input file '%s' does not exist\n", *inputFile)
		os.Exit(1)
	}

	// Create or open Excel file
	var excel *excelize.File
	var rowCount int
	var existingRecords = make(map[string]bool)

	// Check if output Excel file exists
	if _, err := os.Stat(*outputFile); os.IsNotExist(err) {
		// Create new Excel file
		excel = excelize.NewFile()

		// Create default sheet
		excel.SetSheetName("Sheet1", "Mark Data")

		// Set headers
		excel.SetCellValue("Mark Data", "A1", "Mark Date")
		excel.SetCellValue("Mark Data", "B1", "Mark Time")
		excel.SetCellValue("Mark Data", "C1", "Mark Data")

		// Add some formatting
		styleID, _ := excel.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#DDEBF7"}, Pattern: 1},
			Border: []excelize.Border{
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "left", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})
		excel.SetCellStyle("Mark Data", "A1", "C1", styleID)

		// Start from row 2 for data
		rowCount = 2
	} else {
		// Open existing Excel file
		var err error
		excel, err = excelize.OpenFile(*outputFile)
		if err != nil {
			fmt.Printf("Error opening Excel file: %v\n", err)
			os.Exit(1)
		}

		// Read existing records to avoid duplicates
		rows, err := excel.GetRows("Mark Data")
		if err != nil {
			fmt.Printf("Error reading Excel rows: %v\n", err)
			os.Exit(1)
		}

		// Skip header row
		for i := 1; i < len(rows); i++ {
			if len(rows[i]) >= 3 {
				// Create a unique key for each record
				key := fmt.Sprintf("%s|%s|%s", rows[i][0], rows[i][1], rows[i][2])
				existingRecords[key] = true
			}
		}

		// Set row count for appending
		rowCount = len(rows) + 1
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(*outputFile)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Starting to monitor file: %s\n", *inputFile)
	fmt.Printf("Writing to Excel file: %s\n", *outputFile)
	fmt.Printf("Loaded %d existing records to avoid duplicates\n", len(existingRecords))
	fmt.Println("Press Ctrl+C to stop")

	// Keep track of the last position we read to
	var lastPos int64 = 0

	// Regular expression to parse the mark data format
	markRegex := regexp.MustCompile(`&\[\((.*?) (.*?)\)\n(.*?)\n&\]`)

	for {
		// Open the file
		file, err := os.Open(*inputFile)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Get current file size
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Printf("Error getting file info: %v\n", err)
			file.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		// If file hasn't changed, wait and try again
		if fileInfo.Size() <= lastPos {
			file.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Seek to where we last read
		_, err = file.Seek(lastPos, io.SeekStart)
		if err != nil {
			fmt.Printf("Error seeking in file: %v\n", err)
			file.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		// Read the new content
		scanner := bufio.NewScanner(file)
		var buffer strings.Builder

		for scanner.Scan() {
			line := scanner.Text() + "\n"
			buffer.WriteString(line)
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			file.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		// Update last position
		lastPos = fileInfo.Size()

		// Process the content
		content := buffer.String()
		matches := markRegex.FindAllStringSubmatch(content, -1)

		if len(matches) > 0 {
			newRecordsCount := 0

			for _, match := range matches {
				if len(match) >= 4 {
					markDate := match[1]
					markTime := match[2]
					markData := strings.TrimSpace(match[3])

					// Create a unique key for this record
					recordKey := fmt.Sprintf("%s|%s|%s", markDate, markTime, markData)

					// Check if record already exists
					if _, exists := existingRecords[recordKey]; !exists {
						// New record, add to Excel
						excel.SetCellValue("Mark Data", fmt.Sprintf("A%d", rowCount), markDate)
						excel.SetCellValue("Mark Data", fmt.Sprintf("B%d", rowCount), markTime)
						excel.SetCellValue("Mark Data", fmt.Sprintf("C%d", rowCount), markData)

						// Add to tracking map
						existingRecords[recordKey] = true

						// Increment counters
						rowCount++
						newRecordsCount++

						fmt.Println("\n--- New Mark Data Added ---")
						fmt.Printf("Mark Date: %s\n", markDate)
						fmt.Printf("Mark Time: %s\n", markTime)
						fmt.Printf("Mark Data: %s\n", markData)
						fmt.Println("----------------------------")
					} else {
						fmt.Println("\n--- Duplicate Record Skipped ---")
						fmt.Printf("Mark Date: %s\n", markDate)
						fmt.Printf("Mark Time: %s\n", markTime)
						fmt.Printf("Mark Data: %s\n", markData)
						fmt.Println("--------------------------------")
					}
				}
			}

			// Save Excel file if we added any new records
			if newRecordsCount > 0 {
				if err := excel.SaveAs(*outputFile); err != nil {
					fmt.Printf("Error saving Excel file: %v\n", err)
				} else {
					fmt.Printf("Successfully updated Excel file with %d new entries\n", newRecordsCount)
				}
			} else {
				fmt.Println("No new unique records to add to Excel file")
			}
		}

		file.Close()
		time.Sleep(500 * time.Millisecond)
	}
}
