package localtools

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"
)

// ImageDimensions mirrors TS ImageDimensions for coordinate mapping.
type ImageDimensions struct {
	OriginalWidth  int `json:"originalWidth,omitempty"`
	OriginalHeight int `json:"originalHeight,omitempty"`
	DisplayWidth   int `json:"displayWidth,omitempty"`
	DisplayHeight  int `json:"displayHeight,omitempty"`
}

// ImageResult mirrors the TS ImageResult type.
type ImageResult struct {
	Type string `json:"type"`
	File struct {
		Base64       string            `json:"base64"`
		Type         string            `json:"type"`
		OriginalSize int               `json:"originalSize"`
		Dimensions   *ImageDimensions  `json:"dimensions,omitempty"`
	} `json:"file"`
}

// ImageTokenBudgetError is returned when image exceeds token budget even after compression.
type ImageTokenBudgetError struct {
	EstimatedTokens int
	MaxTokens       int
}

func (e *ImageTokenBudgetError) Error() string {
	return fmt.Sprintf("image exceeds token budget (~%d tokens estimated; limit %d). Cannot compress further.", e.EstimatedTokens, e.MaxTokens)
}

var errEmptyImage = errors.New("image file is empty (0 bytes)")

// detectImageFormatFromBuffer mirrors TS detectImageFormatFromBuffer.
func detectImageFormatFromBuffer(buf []byte) string {
	if len(buf) < 4 {
		return "png"
	}
	// PNG: 89 50 4E 47
	if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return "png"
	}
	// JPEG: FF D8 FF
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return "jpeg"
	}
	// GIF: 47 49 46
	if buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46 {
		return "gif"
	}
	// WebP: RIFF .... WEBP
	if buf[0] == 0x52 && buf[1] == 0x49 && buf[2] == 0x46 && buf[3] == 0x46 {
		if len(buf) >= 12 && buf[8] == 0x57 && buf[9] == 0x45 && buf[10] == 0x42 && buf[11] == 0x50 {
			return "webp"
		}
	}
	return "png"
}

// decodeImage decodes an image buffer, returning the image and its format.
// For WebP (not supported by Go stdlib), returns a special error so caller
// falls back to format conversion.
func decodeImage(buf []byte) (image.Image, string, error) {
	// Try WebP detection first — Go stdlib can't decode WebP
	detected := detectImageFormatFromBuffer(buf)
	if detected == "webp" {
		return nil, "webp", fmt.Errorf("webp format requires conversion")
	}
	img, fmtName, err := image.Decode(newBytesReadSeeker(buf))
	if err != nil {
		return nil, "", err
	}
	return img, fmtName, nil
}

// bytesReadSeeker wraps a byte slice as io.ReadSeeker.
type bytesReadSeeker struct {
	data []byte
	pos  int
}

func newBytesReadSeeker(data []byte) *bytesReadSeeker {
	return &bytesReadSeeker{data: data}
}

func (r *bytesReadSeeker) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = int64(r.pos) + offset
	case io.SeekEnd:
		abs = int64(len(r.data)) + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}
	r.pos = int(abs)
	return abs, nil
}

// resizeImage resizes an image to fit within maxWidth x maxHeight while
// maintaining aspect ratio (without enlargement). Uses nearest-neighbor
// scaling for simplicity (always available in Go stdlib).
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= maxWidth && h <= maxHeight {
		return img
	}
	ratio := math.Min(float64(maxWidth)/float64(w), float64(maxHeight)/float64(h))
	ratio = math.Min(ratio, 1.0) // never enlarge
	nw := int(math.Round(float64(w) * ratio))
	nh := int(math.Round(float64(h) * ratio))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return scaleImageNearest(img, nw, nh)
}

// scaleImageNearest performs nearest-neighbor scaling.
func scaleImageNearest(src image.Image, dstW, dstH int) image.Image {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := 0; dy < dstH; dy++ {
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			sy := dy * srcH / dstH
			dst.Set(dx, dy, src.At(srcBounds.Min.X+sx, srcBounds.Min.Y+sy))
		}
	}
	return dst
}

// encodeJPEGQuality encodes image as JPEG with the given quality (1-100).
func encodeJPEGQuality(img image.Image, quality int) ([]byte, error) {
	var buf strings.Builder
	err := jpeg.Encode(asWriter(&buf), img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// encodePNGBestCompression encodes image as PNG with best compression.
func encodePNGBestCompression(img image.Image) ([]byte, error) {
	var buf strings.Builder
	err := png.Encode(asWriter(&buf), img)
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

type stringWriter struct {
	b *strings.Builder
}

func asWriter(b *strings.Builder) *stringWriter {
	return &stringWriter{b: b}
}

func (w *stringWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

// recompressAsJPEG re-encodes image as JPEG with progressive quality reduction
// until it fits within targetBytes. Returns the best result.
func recompressAsJPEG(img image.Image, targetBytes int) ([]byte, string, error) {
	qualities := []int{85, 70, 50, 30, 15}
	for _, q := range qualities {
		data, err := encodeJPEGQuality(img, q)
		if err != nil {
			continue
		}
		if len(data) <= targetBytes {
			return data, "jpeg", nil
		}
	}
	// Last resort: smallest quality
	data, err := encodeJPEGQuality(img, 10)
	if err != nil {
		return nil, "", err
	}
	return data, "jpeg", nil
}

// recompressAsPNG re-encodes image as PNG with best compression.
func recompressAsPNG(img image.Image, targetBytes int) ([]byte, string, error) {
	data, err := encodePNGBestCompression(img)
	if err != nil {
		return nil, "", err
	}
	if len(data) <= targetBytes {
		return data, "png", nil
	}
	return nil, "", fmt.Errorf("png compression to %d bytes failed: got %d", targetBytes, len(data))
}

// compressImageWithTokenBudget mirrors TS compressImageBufferWithTokenLimit.
// Converts token limit to byte limit: maxBytes = floor((maxTokens / 0.125) * 0.75).
func compressImageWithTokenBudget(img image.Image, format string, maxTokens int, maxBytes int) ([]byte, string, error) {
	targetBytes := int(math.Floor(float64(maxTokens) / 0.125 * 0.75))
	if maxBytes > 0 && maxBytes < targetBytes {
		targetBytes = maxBytes
	}

	// Try format-preserving compression first
	switch format {
	case "png":
		if data, mt, err := recompressAsPNG(img, targetBytes); err == nil {
			return data, mt, nil
		}
		// PNG too large, try JPEG
		if data, mt, err := recompressAsJPEG(img, targetBytes); err == nil {
			return data, mt, nil
		}
	case "jpeg", "jpg":
		if data, mt, err := recompressAsJPEG(img, targetBytes); err == nil {
			return data, mt, nil
		}
	case "gif":
		// GIF → JPEG conversion
		if data, mt, err := recompressAsJPEG(img, targetBytes); err == nil {
			return data, mt, nil
		}
	case "webp":
		// WebP → JPEG conversion (can't decode webp, so fall through)
		if data, mt, err := recompressAsJPEG(img, targetBytes); err == nil {
			return data, mt, nil
		}
	default:
		if data, mt, err := recompressAsJPEG(img, targetBytes); err == nil {
			return data, mt, nil
		}
	}

	return nil, "", fmt.Errorf("cannot compress image to fit token budget")
}

// readImageWithTokenBudget mirrors TS readImageWithTokenBudget.
// Reads an image file, applies resize/compression to fit within token budget.
func readImageWithTokenBudget(absPath string, maxTokens int) (*ImageResult, error) {
	maxImageReadBytes := 50 << 20 // 50MB cap to prevent OOM
	buf, err := readFileSizeLimited(absPath, maxImageReadBytes)
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, fmt.Errorf("%w: %s", errEmptyImage, absPath)
	}
	originalSize := len(buf)
	mediaType := "image/" + detectImageFormatFromBuffer(buf)

	// Try to decode and process the image
	img, fmtName, decErr := decodeImage(buf)
	if decErr != nil {
		// WebP or unsupported format: return raw base64 with note
		return createRawImageResult(buf, mediaType, originalSize)
	}

	// Apply standard resize
	resized := resizeImage(img, 2000, 2000)
	var processedBuf []byte
	var processedType string

	// Encode resized image
	switch fmtName {
	case "png":
		processedBuf, err = encodePNGBestCompression(resized)
		processedType = "png"
	case "jpeg", "jpg":
		processedBuf, err = encodeJPEGQuality(resized, 85)
		processedType = "jpeg"
	default:
		processedBuf, err = encodeJPEGQuality(resized, 85)
		processedType = "jpeg"
	}
	if err != nil {
		return createRawImageResult(buf, mediaType, originalSize)
	}

	// Check token budget
	estimatedTokens := estimateImageTokens(len(processedBuf))
	if estimatedTokens <= maxTokens {
		imgType := "image/" + processedType
		out := &ImageResult{Type: "image"}
		out.File.Base64 = base64.StdEncoding.EncodeToString(processedBuf)
		out.File.Type = imgType
		out.File.OriginalSize = originalSize
		out.File.Dimensions = getImageDimensions(img, resized)
		return out, nil
	}

	// Over budget: try aggressive compression from original
	compressed, compType, compErr := compressImageWithTokenBudget(resized, fmtName, maxTokens, 0)
	if compErr != nil {
		// Failed to compress enough: return raw
		return createRawImageResult(buf, mediaType, originalSize)
	}

	out := &ImageResult{Type: "image"}
	out.File.Base64 = base64.StdEncoding.EncodeToString(compressed)
	out.File.Type = "image/" + compType
	out.File.OriginalSize = originalSize
	out.File.Dimensions = getImageDimensions(img, resized)
	return out, nil
}

func getImageDimensions(original, display image.Image) *ImageDimensions {
	ob := original.Bounds()
	db := display.Bounds()
	dims := &ImageDimensions{
		OriginalWidth:  ob.Dx(),
		OriginalHeight: ob.Dy(),
		DisplayWidth:   db.Dx(),
		DisplayHeight:  db.Dy(),
	}
	if dims.OriginalWidth == dims.DisplayWidth && dims.OriginalHeight == dims.DisplayHeight {
		return dims // Still return dimensions when they match
	}
	return dims
}

// estimateImageTokens mirrors TS: estimatedTokens = Math.ceil(base64.length * 0.125).
func estimateImageTokens(dataLen int) int {
	base64Len := int(math.Ceil(float64(dataLen) * 4.0 / 3.0))
	return int(math.Ceil(float64(base64Len) * 0.125))
}

func createRawImageResult(buf []byte, mediaType string, originalSize int) (*ImageResult, error) {
	out := &ImageResult{Type: "image"}
	out.File.Base64 = base64.StdEncoding.EncodeToString(buf)
	out.File.Type = mediaType
	out.File.OriginalSize = originalSize
	// PNG header dimension detection for oversized checks
	if len(buf) >= 24 && buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		w := int(buf[16])<<24 | int(buf[17])<<16 | int(buf[18])<<8 | int(buf[19])
		h := int(buf[20])<<24 | int(buf[21])<<16 | int(buf[22])<<8 | int(buf[23])
		if w > 0 && h > 0 {
			out.File.Dimensions = &ImageDimensions{
				OriginalWidth:  w,
				OriginalHeight: h,
				DisplayWidth:   w,
				DisplayHeight:  h,
			}
		}
	}
	return out, nil
}
