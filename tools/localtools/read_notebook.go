package localtools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// readFileSizeLimited reads a file up to maxBytes. Defined locally to avoid
// import cycle with the tools package.
func readFileSizeLimited(absPath string, maxBytes int) ([]byte, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf, err := io.ReadAll(io.LimitReader(f, int64(maxBytes+1)))
	if err != nil {
		return nil, err
	}
	if len(buf) > maxBytes {
		return nil, fmt.Errorf("file content (%d bytes) exceeds maximum allowed size (%d bytes). Use offset and limit parameters to read specific portions of the file, or search for specific content instead of reading the whole file", len(buf), maxBytes)
	}
	return buf, nil
}

// processNotebookCell mirrors TS processCell in src/utils/notebook.ts.
// Transforms a raw Notebook cell into a processed NotebookCellSource with
// extracted metadata, truncated output text, and image detection.
func processNotebookCell(cell map[string]any, index int, codeLanguage string, includeLargeOutputs bool) map[string]any {
	cellType, _ := cell["cell_type"].(string)
	source := extractCellSource(cell)
	executionCount := extractExecutionCount(cell)
	cellID := extractCellID(cell, index)

	result := map[string]any{
		"cellType": cellType,
		"source":   source,
		"cell_id":  cellID,
	}

	if cellType == "code" {
		result["language"] = codeLanguage
		if ec := executionCount; ec > 0 {
			result["execution_count"] = ec
		}
	}

	// Process outputs for code cells
	if cellType == "code" {
		if rawOutputs, ok := cell["outputs"].([]any); ok && len(rawOutputs) > 0 {
			processed := make([]any, 0, len(rawOutputs))
			for _, ro := range rawOutputs {
				if om, ok := ro.(map[string]any); ok {
					processed = append(processed, processNotebookOutput(om))
				}
			}
			if !includeLargeOutputs && isLargeOutputs(processed) {
				result["outputs"] = []map[string]any{
					{
						"output_type": "stream",
						"text":        fmt.Sprintf("Outputs are too large to include. Use Bash tool with: cat <notebook_path> | jq '.cells[%d].outputs'", index),
					},
				}
			} else if len(processed) > 0 {
				result["outputs"] = processed
			}
		}
	}

	return result
}

const largeOutputThreshold = 10000

func isLargeOutputs(outputs []any) bool {
	var size int
	for _, o := range outputs {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := m["text"].(string); ok {
			size += len(t)
		}
		if img, ok := m["image"].(map[string]any); ok {
			if d, ok := img["image_data"].(string); ok {
				size += len(d)
			}
		}
		if size > largeOutputThreshold {
			return true
		}
	}
	return false
}

func processNotebookOutput(output map[string]any) map[string]any {
	ot, _ := output["output_type"].(string)
	switch ot {
	case "stream":
		text := processOutputText(output["text"])
		return map[string]any{
			"output_type": ot,
			"text":        text,
		}
	case "execute_result", "display_data":
		r := map[string]any{
			"output_type": ot,
			"text":        processOutputText(extractPlainText(output["data"])),
		}
		if img := extractNotebookImage(output["data"]); img != nil {
			r["image"] = img
		}
		return r
	case "error":
		ename, _ := output["ename"].(string)
		evalue, _ := output["evalue"].(string)
		traceback := extractTraceback(output["traceback"])
		return map[string]any{
			"output_type": ot,
			"text":        ename + ": " + evalue + "\n" + strings.Join(traceback, "\n"),
		}
	default:
		return output
	}
}

// processOutputText mirrors TS processOutputText: joins string arrays, formats output.
func processOutputText(text any) string {
	switch v := text.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, s := range v {
			if sv, ok := s.(string); ok {
				parts = append(parts, sv)
			}
		}
		return strings.Join(parts, "")
	case []string:
		return strings.Join(v, "")
	default:
		return fmt.Sprintf("%v", text)
	}
}

func extractPlainText(data any) string {
	m, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	return processOutputText(m["text/plain"])
}

func extractNotebookImage(data any) map[string]any {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	if png, ok := m["image/png"].(string); ok {
		return map[string]any{
			"image_data": strings.ReplaceAll(png, " ", ""),
			"media_type": "image/png",
		}
	}
	if jpeg, ok := m["image/jpeg"].(string); ok {
		return map[string]any{
			"image_data": strings.ReplaceAll(jpeg, " ", ""),
			"media_type": "image/jpeg",
		}
	}
	return nil
}

func extractTraceback(tb any) []string {
	switch v := tb.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if sv, ok := s.(string); ok {
				out = append(out, sv)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func extractCellSource(cell map[string]any) string {
	if src, ok := cell["source"].(string); ok {
		return src
	}
	if srcArr, ok := cell["source"].([]any); ok {
		var parts []string
		for _, s := range srcArr {
			if sv, ok := s.(string); ok {
				parts = append(parts, sv)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

func extractExecutionCount(cell map[string]any) int {
	if ec, ok := cell["execution_count"]; ok {
		if f, ok := ec.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func extractCellID(cell map[string]any, fallbackIndex int) string {
	if id, ok := cell["id"].(string); ok && id != "" {
		return id
	}
	if meta, ok := cell["metadata"].(map[string]any); ok {
		if id, ok := meta["id"].(string); ok && id != "" {
			return id
		}
	}
	return fmt.Sprintf("cell-%d", fallbackIndex)
}

// readNotebookProcessed mirrors TS readNotebook in src/utils/notebook.ts.
// Reads a notebook JSON file and returns processed cell data.
func readNotebookProcessed(absPath string, maxSizeBytes int) ([]map[string]any, error) {
	raw, err := readFileSizeLimited(absPath, maxSizeBytes)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("notebook is not valid JSON: %w", err)
	}
	rawCells, ok := root["cells"].([]any)
	if !ok {
		return nil, fmt.Errorf("notebook missing cells array")
	}
	language := "python"
	if md, ok := root["metadata"].(map[string]any); ok {
		if li, ok := md["language_info"].(map[string]any); ok {
			if name, ok := li["name"].(string); ok && name != "" {
				language = name
			}
		}
	}
	cells := make([]map[string]any, 0, len(rawCells))
	for i, rc := range rawCells {
		if cm, ok := rc.(map[string]any); ok {
			cells = append(cells, processNotebookCell(cm, i, language, false))
		}
	}
	return cells, nil
}

