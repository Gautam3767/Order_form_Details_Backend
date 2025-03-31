package services

import (
	"bytes"   // For capturing command output
	"context" // For command timeouts (optional but good practice)
	"errors"  // For checking specific errors like command not found
	"fmt"     // For formatting error messages
	"io"      // For handling input stream (the PDF file)
	"log"     // For logging errors or warnings
	"os/exec" // For running external commands (pdftotext)
	"strings" // For trimming whitespace from the result
	"time"    // For setting command timeout
)

// pdfTimeout defines how long we wait for the pdftotext command to run.
const pdfTimeout = 15 * time.Second

// ExtractTextFromPDF uses the external 'pdftotext' command-line tool
// to extract text content from a given PDF data stream.
//
// IMPORTANT: Requires 'pdftotext' (part of the poppler-utils package)
// to be installed and accessible in the system's PATH.
// - Ubuntu/Debian: sudo apt-get update && sudo apt-get install poppler-utils
// - macOS (Homebrew): brew install poppler
// - Windows: Requires installing poppler, potentially via scoop, chocolatey, or manual download.
//
// Args:
//
//	pdfStream: An io.Reader providing the raw PDF data.
//
// Returns:
//
//	string: The extracted text content.
//	error: An error if pdftotext fails, isn't found, or times out.
func ExtractTextFromPDF(pdfStream io.Reader) (string, error) {
	// Create a context with a timeout to prevent the command from running indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), pdfTimeout)
	defer cancel() // Ensure context resources are released

	// Prepare the command: pdftotext <input> <output>
	// Using "-" for input means read from stdin.
	// Using "-" for output means write text to stdout.
	cmd := exec.CommandContext(ctx, "pdftotext", "-", "-")

	// Set the standard input for the command to our PDF stream.
	cmd.Stdin = pdfStream

	// Prepare buffers to capture the command's standard output (the extracted text)
	// and standard error (any error messages from pdftotext).
	var outbuf bytes.Buffer
	var errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	log.Println("Attempting to run pdftotext...") // Log attempt

	// Execute the command.
	err := cmd.Run()

	// Check if the context timed out or was cancelled.
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("pdftotext command timed out after %v", pdfTimeout)
		return "", fmt.Errorf("pdftotext command timed out after %v", pdfTimeout)
	}

	// Check for errors during command execution.
	if err != nil {
		stderrOutput := errbuf.String()
		log.Printf("pdftotext execution failed. Stderr: %s", stderrOutput)

		// Check specifically if the error is because the command wasn't found.
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("pdftotext command not found: please ensure poppler-utils is installed and in the system PATH")
		}

		// Return a generic error including the original error and stderr output.
		return "", fmt.Errorf("pdftotext execution failed: %w, stderr: %s", err, stderrOutput)
	}

	// If execution was successful, extract the text from the output buffer.
	extractedText := strings.TrimSpace(outbuf.String())
	log.Printf("pdftotext executed successfully. Extracted %d bytes of text.", len(extractedText))

	// Even if the command ran, it might not have output anything (e.g., image-only PDF).
	if extractedText == "" {
		log.Println("Warning: pdftotext ran successfully but produced no text output. PDF might be image-based or empty.")
	}

	return extractedText, nil
}
