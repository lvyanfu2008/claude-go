package messagesapi

import "fmt"

// User-facing strings for stripTargets (src/api-client/userFacingAttachmentErrorMessages.ts).
func pdfTooLargeErrorMessage(nonInteractive bool) string {
	limits := fmt.Sprintf("max %d pages, %s", apiPDFMaxPages, formatFileSize(pdfTargetRawSize))
	if nonInteractive {
		return fmt.Sprintf("PDF too large (%s). Try reading the file a different way (e.g., extract text with pdftotext).", limits)
	}
	return fmt.Sprintf("PDF too large (%s). Double press esc to go back and try again, or use pdftotext to convert to text first.", limits)
}

func pdfPasswordProtectedErrorMessage(nonInteractive bool) string {
	if nonInteractive {
		return "PDF is password protected. Try using a CLI tool to extract or convert the PDF."
	}
	return "PDF is password protected. Please double press esc to edit your message and try again."
}

func pdfInvalidErrorMessage(nonInteractive bool) string {
	if nonInteractive {
		return "The PDF file was not valid. Try converting it to text first (e.g., pdftotext)."
	}
	return "The PDF file was not valid. Double press esc to go back and try again with a different file."
}

func imageTooLargeErrorMessage(nonInteractive bool) string {
	if nonInteractive {
		return "Image was too large. Try resizing the image or using a different approach."
	}
	return "Image was too large. Double press esc to go back and try again with a smaller image."
}

func requestTooLargeErrorMessage(nonInteractive bool) string {
	limits := fmt.Sprintf("max %s", formatFileSize(pdfTargetRawSize))
	if nonInteractive {
		return fmt.Sprintf("Request too large (%s). Try with a smaller file.", limits)
	}
	return fmt.Sprintf("Request too large (%s). Double press esc to go back and try with a smaller file.", limits)
}

func errorToBlockTypes(nonInteractive bool) map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		pdfTooLargeErrorMessage(nonInteractive):            {"document": {}},
		pdfPasswordProtectedErrorMessage(nonInteractive):   {"document": {}},
		pdfInvalidErrorMessage(nonInteractive):             {"document": {}},
		imageTooLargeErrorMessage(nonInteractive):          {"image": {}},
		requestTooLargeErrorMessage(nonInteractive):        {"document": {}, "image": {}},
		// Also register the alternate branch so persisted transcripts match either mode.
		pdfTooLargeErrorMessage(!nonInteractive):          {"document": {}},
		pdfPasswordProtectedErrorMessage(!nonInteractive): {"document": {}},
		pdfInvalidErrorMessage(!nonInteractive):           {"document": {}},
		imageTooLargeErrorMessage(!nonInteractive):        {"image": {}},
		requestTooLargeErrorMessage(!nonInteractive):      {"document": {}, "image": {}},
	}
}
