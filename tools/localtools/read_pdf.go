package localtools

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PDF constants mirror src/constants/apiLimits.ts.
const (
	pdfTargetRawSize          = 20 * 1024 * 1024  // 20 MB — max PDF for full document block
	pdfExtractSizeThreshold   = 3 * 1024 * 1024   // 3 MB — threshold to prefer pdftoppm over full read
	pdfMaxExtractSize         = 100 * 1024 * 1024 // 100 MB — max PDF for pdftoppm extraction
	pdfAtMentionInlineThreshold = 10               // pages — max pages before requiring explicit range
)

// ReadPDFFullDocument reads a PDF as base64-encoded data for document block API usage.
// Mirrors src/utils/pdf.ts readPDF.
// Returns (base64Encoded, mediaType, originalSize, error).
func ReadPDFFullDocument(absPath string) ([]byte, string, int, error) {
	st, err := os.Stat(absPath)
	if err != nil {
		return nil, "", 0, err
	}
	originalSize := int(st.Size())

	if originalSize == 0 {
		return nil, "", 0, fmt.Errorf("PDF file is empty: %s", absPath)
	}
	if originalSize > pdfTargetRawSize {
		return nil, "", 0, fmt.Errorf("PDF file exceeds maximum allowed size of 20 MB")
	}

	buf, err := os.ReadFile(absPath)
	if err != nil {
		return nil, "", 0, err
	}

	// Validate PDF magic bytes — reject non-PDF files before they enter conversation context.
	if len(buf) < 5 || string(buf[:5]) != "%PDF-" {
		return nil, "", 0, fmt.Errorf("File is not a valid PDF (missing %%PDF- header): %s", absPath)
	}

	encoded := base64.StdEncoding.EncodeToString(buf)
	return []byte(encoded), "application/pdf", originalSize, nil
}

// ExtractPDFPages extracts PDF pages as JPEG images via pdftoppm.
// Mirrors src/utils/pdf.ts extractPDFPages.
// Returns (outputDir, pageCount, error).
func ExtractPDFPages(absPath string, firstPage, lastPage int) (outputDir string, pageCount int, err error) {
	st, err := os.Stat(absPath)
	if err != nil {
		return "", 0, err
	}
	if st.Size() == 0 {
		return "", 0, fmt.Errorf("PDF file is empty: %s", absPath)
	}
	if st.Size() > pdfMaxExtractSize {
		return "", 0, fmt.Errorf("PDF file exceeds maximum allowed size for page extraction (100 MB)")
	}

	// Check pdftoppm availability
	if !isPdftoppmAvailable() {
		return "", 0, fmt.Errorf("pdftoppm is not installed. Install poppler-utils (e.g. `brew install poppler` or `apt-get install poppler-utils`) to enable PDF page rendering")
	}

	// Create temp output directory
	outputDir, err = os.MkdirTemp("", "pdf-pages-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// pdftoppm produces files like <prefix>-01.jpg, <prefix>-02.jpg, etc.
	prefix := filepath.Join(outputDir, "page")
	args := []string{"-jpeg", "-r", "100"}
	if firstPage > 0 {
		args = append(args, "-f", strconv.Itoa(firstPage))
	}
	if lastPage > 0 {
		args = append(args, "-l", strconv.Itoa(lastPage))
	}
	args = append(args, absPath, prefix)

	cmd := exec.Command("pdftoppm", args...)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := stderrBuf.String()
		// Clean up temp dir on failure
		os.RemoveAll(outputDir)

		if regexp.MustCompile(`(?i)password`).MatchString(stderr) {
			return "", 0, fmt.Errorf("PDF is password-protected. Please provide an unprotected version")
		}
		if regexp.MustCompile(`(?i)damaged|corrupt|invalid`).MatchString(stderr) {
			return "", 0, fmt.Errorf("PDF file is corrupted or invalid")
		}
		return "", 0, fmt.Errorf("pdftoppm failed: %s", stderr)
	}

	// Count generated JPEG files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		os.RemoveAll(outputDir)
		return "", 0, fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jpg") {
			pageCount++
		}
	}

	if pageCount == 0 {
		os.RemoveAll(outputDir)
		return "", 0, fmt.Errorf("pdftoppm produced no output pages. The PDF may be invalid")
	}

	return outputDir, pageCount, nil
}

var pdftoppmChecked bool
var pdftoppmOK bool

// isPdftoppmAvailable checks whether pdftoppm binary is available (cached).
// Mirrors src/utils/pdf.ts isPdftoppmAvailable.
func isPdftoppmAvailable() bool {
	if pdftoppmChecked {
		return pdftoppmOK
	}
	pdftoppmChecked = true
	cmd := exec.Command("pdftoppm", "-v")
	out, err := cmd.CombinedOutput()
	// pdftoppm prints version info to stderr and exits 0 (or sometimes non-zero on older versions)
	pdftoppmOK = err == nil || len(out) > 0
	return pdftoppmOK
}

// GetPDFPageCount gets page count via pdfinfo (from poppler-utils).
// Mirrors src/utils/pdf.ts getPDFPageCount.
func GetPDFPageCount(absPath string) (int, error) {
	cmd := exec.Command("pdfinfo", absPath)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo failed: %w", err)
	}
	re := regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)
	m := re.FindStringSubmatch(string(out))
	if m == nil {
		return 0, fmt.Errorf("could not determine page count")
	}
	count, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("invalid page count: %s", m[1])
	}
	return count, nil
}

// shouldExtractPages determines whether to use pdftoppm extraction vs full document block.
// Mirrors TS logic: extract if !isPDFSupported() or file > PDF_EXTRACT_SIZE_THRESHOLD.
func shouldExtractPDFPages(absPath string) bool {
	st, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return st.Size() > pdfExtractSizeThreshold || !isPdftoppmAvailable()
}
