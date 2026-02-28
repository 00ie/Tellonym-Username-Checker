package checker

import (
	"strings"
	"testing"
)

func TestNormalizeUsernameRulesBoundaries(t *testing.T) {
	rules := NormalizeUsernameRules(UsernameRules{
		MinLength:          1,
		MaxLength:          200,
		MaxConsecutiveDots: 0,
	})

	if rules.MinLength != 3 {
		t.Fatalf("expected MinLength=3, got %d", rules.MinLength)
	}
	if rules.MaxLength != 30 {
		t.Fatalf("expected MaxLength=30, got %d", rules.MaxLength)
	}
	if rules.MaxConsecutiveDots != 1 {
		t.Fatalf("expected MaxConsecutiveDots=1, got %d", rules.MaxConsecutiveDots)
	}
	if !rules.AllowLetters || !rules.AllowNumbers {
		t.Fatalf("expected letters and numbers enabled fallback")
	}
}

func TestUsernameGeneratorRespectsRules(t *testing.T) {
	rules := UsernameRules{
		MinLength:           3,
		MaxLength:           30,
		AllowLetters:        true,
		AllowNumbers:        true,
		AllowUnderscore:     true,
		AllowDot:            true,
		DisallowLeadingDot:  true,
		DisallowTrailingDot: true,
		MaxConsecutiveDots:  1,
	}

	generator := NewUsernameGenerator(12, rules)
	normalized := NormalizeUsernameRules(rules)

	for i := 0; i < 500; i++ {
		value := generator.Generate()
		if len(value) != 12 {
			t.Fatalf("expected username length 12, got %d (%s)", len(value), value)
		}

		ok, reason := normalized.Validate(value)
		if !ok {
			t.Fatalf("generated invalid username: %s (%s)", value, reason)
		}

		if strings.HasPrefix(value, ".") {
			t.Fatalf("generated username with leading dot: %s", value)
		}
		if strings.HasSuffix(value, ".") {
			t.Fatalf("generated username with trailing dot: %s", value)
		}
		if strings.Contains(value, "..") {
			t.Fatalf("generated username with repeated dots: %s", value)
		}
	}
}

func TestUsernameGeneratorClampsLength(t *testing.T) {
	rules := DefaultUsernameRules()

	short := NewUsernameGenerator(1, rules).Generate()
	if len(short) != 3 {
		t.Fatalf("expected clamped short length=3, got %d", len(short))
	}

	long := NewUsernameGenerator(90, rules).Generate()
	if len(long) != 30 {
		t.Fatalf("expected clamped long length=30, got %d", len(long))
	}
}
