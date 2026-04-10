// Mirrors src/utils/config.ts PastedContent and ImageDimensions.
package utils

// ImageDimensions mirrors src/utils/imageResizer.ts ImageDimensions.
type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// PastedContent mirrors src/utils/config.ts PastedContent.
type PastedContent struct {
	ID          int              `json:"id"`
	Type        string           `json:"type"` // text | image
	Content     string           `json:"content"`
	MediaType   *string          `json:"mediaType,omitempty"`
	Filename    *string          `json:"filename,omitempty"`
	Dimensions  *ImageDimensions `json:"dimensions,omitempty"`
	SourcePath  *string          `json:"sourcePath,omitempty"`
}
