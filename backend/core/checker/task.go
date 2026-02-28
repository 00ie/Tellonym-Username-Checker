package checker

import (
	"fmt"
	"strings"

	"tellonym-checker/backend/utils/random"
)

type Task struct {
	Username string
	ID       string
	Priority int
	Retries  int
}

type UsernameGenerator struct {
	length  int
	rules   UsernameRules
	charset string
	noDot   string
}

func NewUsernameGenerator(length int, rules UsernameRules) *UsernameGenerator {
	normalized := NormalizeUsernameRules(rules)
	if length < normalized.MinLength {
		length = normalized.MinLength
	}
	if length > normalized.MaxLength {
		length = normalized.MaxLength
	}
	charset := normalized.AllowedCharset()
	noDot := strings.ReplaceAll(charset, ".", "")
	if noDot == "" {
		noDot = "abcdefghijklmnopqrstuvwxyz0123456789_"
	}
	return &UsernameGenerator{length: length, rules: normalized, charset: charset, noDot: noDot}
}

func (g *UsernameGenerator) Generate() string {
	for attempt := 0; attempt < 100; attempt++ {
		candidate := g.generateCandidate()
		if ok, _ := g.rules.Validate(candidate); ok {
			return candidate
		}
	}

	for attempt := 0; attempt < 100; attempt++ {
		candidate := random.StringFromCharset(g.length, g.noDot)
		if ok, _ := g.rules.Validate(candidate); ok {
			return candidate
		}
	}

	return random.String(g.length)
}

func (g *UsernameGenerator) generateCandidate() string {
	if g.length <= 0 {
		return ""
	}

	buffer := make([]byte, g.length)
	firstCharset := g.charset
	lastCharset := g.charset

	if g.rules.DisallowLeadingDot {
		firstCharset = g.noDot
	}
	if g.rules.DisallowTrailingDot {
		lastCharset = g.noDot
	}

	buffer[0] = random.ByteFromCharset(firstCharset)

	for i := 1; i < g.length-1; i++ {
		buffer[i] = random.ByteFromCharset(g.charset)
	}

	if g.length > 1 {
		buffer[g.length-1] = random.ByteFromCharset(lastCharset)
	}

	return string(buffer)
}

func NewTask(username string, retries int) Task {
	return Task{
		Username: username,
		ID:       fmt.Sprintf("%s", random.String(16)),
		Priority: 1,
		Retries:  retries,
	}
}
