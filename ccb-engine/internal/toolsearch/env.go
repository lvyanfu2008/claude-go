package toolsearch

import (
	"os"
	"strings"
)

func envTruthy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envDefinedFalsy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

func envTrim(k string) string {
	return strings.TrimSpace(os.Getenv(k))
}
