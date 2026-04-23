package localtools

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PDF limits mirror src/constants/apiLimits.ts PDF_MAX_PAGES_PER_READ.
const pdfMaxPagesPerRead = 20

// Reserved errors for PDF paths (FileReadTool.ts PDF branch — poppler / document blocks / image pipeline).
// Callers should use [errors.Is] to detect stub paths until implementations land.
var (
	ErrReadPDFPagesNotImplementedInGo = errors.New(
		"go localtools: PDF page extraction not implemented; use TypeScript FileReadTool or add poppler-backed extractPDFPages",
	)
	ErrReadPDFFullNotImplementedInGo = errors.New(
		"go localtools: full PDF read (document block) not implemented; use TypeScript FileReadTool or wire pdf.js-equivalent reader",
	)
)

// ValidateReadPagesParameter mirrors FileReadTool.validateInput pages checks (format + range size) when pages is set.
// Safe to call for any file type — TS validates format even before PDF branch.
func ValidateReadPagesParameter(pages string) error {
	pages = strings.TrimSpace(pages)
	if pages == "" {
		return nil
	}
	first, last, ok := parsePDFPageRange(pages)
	if !ok {
		return fmt.Errorf(
			`Invalid pages parameter: "%s". Use formats like "1-5", "3", or "10-20". Pages are 1-indexed.`,
			pages,
		)
	}
	n := 0
	if last == 0 {
		n = 1
	} else {
		n = last - first + 1
	}
	if n > pdfMaxPagesPerRead {
		return fmt.Errorf(
			`Page range "%s" exceeds maximum of %d pages per request. Please use a smaller range.`,
			pages, pdfMaxPagesPerRead,
		)
	}
	return nil
}

var rePDFPageToken = regexp.MustCompile(`^\s*(\d+)\s*(?:-\s*(\d+)?)?\s*$`)

// parsePDFPageRange returns (first, last, ok). last==0 means single-page form used only as end; caller uses n=1.
// For "1-∞" style TS uses Infinity — we approximate with last = first + pdfMaxPagesPerRead for size check only.
func parsePDFPageRange(s string) (first, last int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	m := rePDFPageToken.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(m[1])
	if err1 != nil || a < 1 {
		return 0, 0, false
	}
	if m[2] == "" {
		return a, 0, true // single page
	}
	b, err2 := strconv.Atoi(m[2])
	if err2 != nil || b < a {
		return 0, 0, false
	}
	return a, b, true
}
