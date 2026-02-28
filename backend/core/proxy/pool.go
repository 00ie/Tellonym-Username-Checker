package proxy

import "strings"

func NormalizeProxyLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		v := strings.TrimSpace(line)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
