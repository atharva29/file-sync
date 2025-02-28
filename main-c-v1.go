package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

func main1() {
	// Define command-line flags
	inputFile := pflag.StringP("input", "i", "", "Path to input file containing mark data")
	pflag.Parse()

	// Validate input
	if *inputFile == "" {
		fmt.Println("Error: Input file path is required")
		fmt.Println("Usage: ./mark-processor --input/-i input_file.txt")
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: Input file '%s' does not exist\n", *inputFile)
		os.Exit(1)
	}

	fmt.Printf("Starting to monitor file: %s\n", *inputFile)
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

		for _, match := range matches {
			if len(match) >= 4 {
				markDate := match[1]
				markTime := match[2]
				markData := strings.TrimSpace(match[3])

				fmt.Println("\n--- New Mark Data Detected ---")
				fmt.Printf("Mark Date: %s\n", markDate)
				fmt.Printf("Mark Time: %s\n", markTime)
				fmt.Printf("Mark Data: %s\n", markData)
				fmt.Println("----------------------------")
			}
		}

		file.Close()
		time.Sleep(500 * time.Millisecond)
	}
}
