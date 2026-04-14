package localtools

import "strconv"

// GrepDefaultHeadLimit mirrors DEFAULT_HEAD_LIMIT in GrepTool/headLimit.ts.
const GrepDefaultHeadLimit = 250

// GrepApplyHeadLimit mirrors applyHeadLimit in headLimit.ts (limit 0 = unlimited).
func GrepApplyHeadLimit[T any](items []T, limit *int, offset int) (sliced []T, appliedLimit *int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(items) {
		return nil, nil
	}
	tail := items[offset:]
	if limit != nil && *limit == 0 {
		return tail, nil
	}
	effective := GrepDefaultHeadLimit
	if limit != nil {
		effective = *limit
	}
	if len(tail) <= effective {
		return tail, nil
	}
	out := make([]T, effective)
	copy(out, tail[:effective])
	return out, &effective
}

// GrepFormatLimitInfo mirrors formatLimitInfo in headLimit.ts.
func GrepFormatLimitInfo(appliedLimit *int, appliedOffset int) string {
	var parts []string
	if appliedLimit != nil {
		parts = append(parts, "limit: "+strconv.Itoa(*appliedLimit))
	}
	if appliedOffset > 0 {
		parts = append(parts, "offset: "+strconv.Itoa(appliedOffset))
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}

func grepPlural(n int, word, pluralWord string) string {
	if n == 1 {
		return word
	}
	return pluralWord
}
