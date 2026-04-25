package processuserinput

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	"goc/types"
)

// ImageDimensions mirrors TS imageResizer.ts ImageDimensions.
type ImageDimensions struct {
	OriginalWidth  int `json:"originalWidth,omitempty"`
	OriginalHeight int `json:"originalHeight,omitempty"`
	DisplayWidth   int `json:"displayWidth,omitempty"`
	DisplayHeight  int `json:"displayHeight,omitempty"`
}

// imageBlockWithDimensions mirrors TS ImageBlockWithDimensions.
type imageBlockWithDimensions struct {
	block      types.ContentBlockParam
	dimensions *ImageDimensions
}

const (
	imageMaxWidth      = 2000
	imageMaxHeight     = 2000
	imageTargetRawSize = 5 * 1024 * 1024 * 3 / 4 // 3.75 MB (3/4 of 5 MB base64 limit, matching TS IMAGE_TARGET_RAW_SIZE)
)

// resizeAndDownsampleImageBlock mirrors TS maybeResizeAndDownsampleImageBlock.
// Accepts base64-encoded image data and media type, returns a resized/compressed
// content block with dimension metadata.
func resizeAndDownsampleImageBlock(data string, mediaType string) (*imageBlockWithDimensions, error) {
	buf, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("image base64 decode: %w", err)
	}
	if len(buf) == 0 {
		// Return as-is with no dimensions (same as TS catch for empty → no resize)
		return &imageBlockWithDimensions{
			block: newImageBlock(data, mediaType),
		}, nil
	}

	originalSize := len(buf)
	format := detectImageFormatFromMagic(buf)
	ext := strings.TrimPrefix(format, "image/")

	// If base64 size is within API limit (5MB raw ≈ ~6.7MB base64) and
	// PNG dimensions aren't oversized, allow through uncompressed
	base64Len := len(data)
	if base64Len <= imageTargetRawSize*4/3 && !isOversizedPNG(buf) {
		// Try to get dimensions even for pass-through
		dims := extractPNGDimensions(buf)
		return &imageBlockWithDimensions{
			block:      newImageBlock(data, format),
			dimensions: dims,
		}, nil
	}

	// Decode image for processing
	img, _, decErr := decodeImageFromBuffer(buf)
	if decErr != nil {
		// Can't decode: return raw data with no dims
		return &imageBlockWithDimensions{
			block: newImageBlock(data, format),
		}, nil
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	if origW == 0 || origH == 0 {
		if originalSize > imageTargetRawSize {
			compressed, err := reencodeJPEG(img, 80)
			if err != nil {
				return rawBlockResult(data, format), nil
			}
			return blockWithImageBuffer(compressed, "jpeg", nil), nil
		}
		return rawBlockResult(data, format), nil
	}

	width, height := origW, origH

	// Original works as-is
	if originalSize <= imageTargetRawSize && width <= imageMaxWidth && height <= imageMaxHeight {
		dims := &ImageDimensions{
			OriginalWidth:  origW,
			OriginalHeight: origH,
			DisplayWidth:   width,
			DisplayHeight:  height,
		}
		return &imageBlockWithDimensions{
			block:      newImageBlock(data, format),
			dimensions: dims,
		}, nil
	}

	needsResize := width > imageMaxWidth || height > imageMaxHeight

	// Dimensions OK but too large: try compression first
	if !needsResize && originalSize > imageTargetRawSize {
		if ext == "png" {
			compressed, err := reencodePNG(img)
			if err == nil && len(compressed) <= imageTargetRawSize {
				dims := &ImageDimensions{
					OriginalWidth:  origW,
					OriginalHeight: origH,
					DisplayWidth:   width,
					DisplayHeight:  height,
				}
				return blockWithImageBuffer(compressed, "png", dims), nil
			}
		}
		for _, q := range []int{80, 60, 40, 20} {
			compressed, err := reencodeJPEG(img, q)
			if err != nil {
				continue
			}
			if len(compressed) <= imageTargetRawSize {
				dims := &ImageDimensions{
					OriginalWidth:  origW,
					OriginalHeight: origH,
					DisplayWidth:   width,
					DisplayHeight:  height,
				}
				return blockWithImageBuffer(compressed, "jpeg", dims), nil
			}
		}
		// Compression alone not enough, fall through to resize
	}

	// Constrain dimensions
	if width > imageMaxWidth {
		height = int(math.Round(float64(height*imageMaxWidth) / float64(width)))
		width = imageMaxWidth
	}
	if height > imageMaxHeight {
		width = int(math.Round(float64(width*imageMaxHeight) / float64(height)))
		height = imageMaxHeight
	}

	resized := resizeNearest(img, width, height)
	resizedBuf, err := encodeWithFormat(resized, ext, 85)
	if err != nil {
		return rawBlockResult(data, format), nil
	}

	// Still too large: try compression stepping
	if len(resizedBuf) > imageTargetRawSize {
		for _, q := range []int{80, 60, 40, 20} {
			compressed, err := reencodeJPEG(resized, q)
			if err != nil {
				continue
			}
			if len(compressed) <= imageTargetRawSize {
				dims := &ImageDimensions{
					OriginalWidth:  origW,
					OriginalHeight: origH,
					DisplayWidth:   width,
					DisplayHeight:  height,
				}
				return blockWithImageBuffer(compressed, "jpeg", dims), nil
			}
		}

		// Last resort: smaller resize + aggressive JPEG
		smallerW := int(math.Min(float64(width), 1000))
		smallerH := int(math.Round(float64(height*smallerW) / float64(width)))
		if smallerH < 1 {
			smallerH = 1
		}
		smaller := resizeNearest(img, smallerW, smallerH)
		ultra, err := reencodeJPEG(smaller, 20)
		if err != nil {
			return rawBlockResult(data, format), nil
		}
		return blockWithImageBuffer(ultra, "jpeg", &ImageDimensions{
			OriginalWidth:  origW,
			OriginalHeight: origH,
			DisplayWidth:   smallerW,
			DisplayHeight:  smallerH,
		}), nil
	}

	return blockWithImageBuffer(resizedBuf, ext, &ImageDimensions{
		OriginalWidth:  origW,
		OriginalHeight: origH,
		DisplayWidth:   width,
		DisplayHeight:  height,
	}), nil
}

// createImageMetadataText mirrors TS createImageMetadataText.
// Returns an image metadata text like:
// "[Image: source: /path, original 1920x1080, displayed at 800x600. Multiply coordinates by 2.40 to map to original image.]"
func createImageMetadataText(dims *ImageDimensions, sourcePath string) string {
	if dims == nil {
		if sourcePath != "" {
			return fmt.Sprintf("[Image source: %s]", sourcePath)
		}
		return ""
	}
	if dims.OriginalWidth <= 0 || dims.OriginalHeight <= 0 || dims.DisplayWidth <= 0 || dims.DisplayHeight <= 0 {
		if sourcePath != "" {
			return fmt.Sprintf("[Image source: %s]", sourcePath)
		}
		return ""
	}

	wasResized := dims.OriginalWidth != dims.DisplayWidth || dims.OriginalHeight != dims.DisplayHeight

	if !wasResized && sourcePath == "" {
		return ""
	}

	var parts []string
	if sourcePath != "" {
		parts = append(parts, fmt.Sprintf("source: %s", sourcePath))
	}

	if wasResized {
		scaleFactor := float64(dims.OriginalWidth) / float64(dims.DisplayWidth)
		parts = append(parts, fmt.Sprintf("original %dx%d, displayed at %dx%d. Multiply coordinates by %.2f to map to original image.",
			dims.OriginalWidth, dims.OriginalHeight, dims.DisplayWidth, dims.DisplayHeight, scaleFactor))
	}

	return fmt.Sprintf("[Image: %s]", strings.Join(parts, ", "))
}

// --- helpers ---

// detectImageFormatFromMagic detects image format from magic bytes.
func detectImageFormatFromMagic(buf []byte) string {
	if len(buf) < 4 {
		return "image/png"
	}
	if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return "image/png"
	}
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return "image/jpeg"
	}
	if buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46 {
		return "image/gif"
	}
	if buf[0] == 0x52 && buf[1] == 0x49 && buf[2] == 0x46 && buf[3] == 0x46 {
		if len(buf) >= 12 && buf[8] == 0x57 && buf[9] == 0x45 && buf[10] == 0x42 && buf[11] == 0x50 {
			return "image/webp"
		}
	}
	return "image/png"
}

// isOversizedPNG checks if a PNG buffer has dimensions exceeding the max.
func isOversizedPNG(buf []byte) bool {
	if len(buf) < 24 {
		return false
	}
	if buf[0] != 0x89 || buf[1] != 0x50 || buf[2] != 0x4E || buf[3] != 0x47 {
		return false
	}
	w := int(buf[16])<<24 | int(buf[17])<<16 | int(buf[18])<<8 | int(buf[19])
	h := int(buf[20])<<24 | int(buf[21])<<16 | int(buf[22])<<8 | int(buf[23])
	return w > imageMaxWidth || h > imageMaxHeight
}

// extractPNGDimensions reads PNG IHDR dimensions from raw bytes.
func extractPNGDimensions(buf []byte) *ImageDimensions {
	if len(buf) < 24 {
		return nil
	}
	if buf[0] != 0x89 || buf[1] != 0x50 || buf[2] != 0x4E || buf[3] != 0x47 {
		return nil
	}
	w := int(buf[16])<<24 | int(buf[17])<<16 | int(buf[18])<<8 | int(buf[19])
	h := int(buf[20])<<24 | int(buf[21])<<16 | int(buf[22])<<8 | int(buf[23])
	if w > 0 && h > 0 {
		return &ImageDimensions{
			OriginalWidth:  w,
			OriginalHeight: h,
			DisplayWidth:   w,
			DisplayHeight:  h,
		}
	}
	return nil
}

// decodeImageFromBuffer decodes an image from a byte buffer using Go stdlib.
func decodeImageFromBuffer(buf []byte) (image.Image, string, error) {
	detected := detectImageFormatFromMagic(buf)
	if detected == "image/webp" {
		return nil, "webp", fmt.Errorf("webp format not supported by Go stdlib")
	}
	img, fmtName, err := image.Decode(&byteReadSeeker{data: buf})
	if err != nil {
		return nil, "", err
	}
	return img, fmtName, nil
}

type byteReadSeeker struct {
	data []byte
	pos  int
}

func (r *byteReadSeeker) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *byteReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = int64(r.pos) + offset
	case io.SeekEnd:
		abs = int64(len(r.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if abs < 0 {
		return 0, fmt.Errorf("negative position")
	}
	r.pos = int(abs)
	return abs, nil
}

// resizeNearest performs nearest-neighbor scaling to target dimensions.
func resizeNearest(src image.Image, dstW, dstH int) image.Image {
	b := src.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := 0; dy < dstH; dy++ {
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			sy := dy * srcH / dstH
			dst.Set(dx, dy, src.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}

// reencodeJPEG re-encodes an image as JPEG with the given quality (1-100).
func reencodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf strings.Builder
	err := jpeg.Encode(&stringWriter{b: &buf}, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// reencodePNG re-encodes an image as PNG with best compression.
func reencodePNG(img image.Image) ([]byte, error) {
	var buf strings.Builder
	err := png.Encode(&stringWriter{b: &buf}, img)
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// encodeWithFormat encodes image preserving format (JPEG/PNG) with fallback to JPEG.
func encodeWithFormat(img image.Image, format string, jpegQuality int) ([]byte, error) {
	switch format {
	case "png":
		return reencodePNG(img)
	default:
		return reencodeJPEG(img, jpegQuality)
	}
}

type stringWriter struct {
	b *strings.Builder
}

func (w *stringWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

// newImageBlock creates a ContentBlockParam with base64 image source.
func newImageBlock(data, mediaType string) types.ContentBlockParam {
	src := map[string]any{
		"type":       "base64",
		"media_type": mediaType,
		"data":       data,
	}
	srcJSON, _ := json.Marshal(src)
	return types.ContentBlockParam{
		Type:   "image",
		Source: json.RawMessage(srcJSON),
	}
}

// blockWithImageBuffer creates an image block from encoded image bytes.
func blockWithImageBuffer(buf []byte, ext string, dims *ImageDimensions) *imageBlockWithDimensions {
	data := base64.StdEncoding.EncodeToString(buf)
	mediaType := "image/" + ext
	return &imageBlockWithDimensions{
		block:      newImageBlock(data, mediaType),
		dimensions: dims,
	}
}

// rawBlockResult creates an image block from raw base64 data (pass-through).
func rawBlockResult(data, mediaType string) *imageBlockWithDimensions {
	return &imageBlockWithDimensions{
		block: newImageBlock(data, mediaType),
	}
}
