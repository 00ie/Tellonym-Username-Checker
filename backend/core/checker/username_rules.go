package checker

import (
	"fmt"
	"strings"
)

type UsernameRules struct {
	MinLength           int  `yaml:"min_length"`
	MaxLength           int  `yaml:"max_length"`
	AllowLetters        bool `yaml:"allow_letters"`
	AllowNumbers        bool `yaml:"allow_numbers"`
	AllowUnderscore     bool `yaml:"allow_underscore"`
	AllowDot            bool `yaml:"allow_dot"`
	DisallowLeadingDot  bool `yaml:"disallow_leading_dot"`
	DisallowTrailingDot bool `yaml:"disallow_trailing_dot"`
	MaxConsecutiveDots  int  `yaml:"max_consecutive_dots"`
}

func DefaultUsernameRules() UsernameRules {
	return UsernameRules{
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
}

func NormalizeUsernameRules(r UsernameRules) UsernameRules {
	if r.MinLength < 3 {
		r.MinLength = 3
	}
	if r.MaxLength <= 0 || r.MaxLength > 30 {
		r.MaxLength = 30
	}
	if r.MaxLength < r.MinLength {
		r.MaxLength = r.MinLength
	}
	if r.MaxConsecutiveDots <= 0 {
		r.MaxConsecutiveDots = 1
	}

	if !r.AllowLetters && !r.AllowNumbers && !r.AllowUnderscore && !r.AllowDot {
		r.AllowLetters = true
		r.AllowNumbers = true
	}

	onlyDot := r.AllowDot && !r.AllowLetters && !r.AllowNumbers && !r.AllowUnderscore
	if onlyDot && (r.DisallowLeadingDot || r.DisallowTrailingDot) {
		r.AllowLetters = true
	}

	return r
}

func (r UsernameRules) AllowedCharset() string {
	n := NormalizeUsernameRules(r)
	charset := ""
	if n.AllowLetters {
		charset += "abcdefghijklmnopqrstuvwxyz"
	}
	if n.AllowNumbers {
		charset += "0123456789"
	}
	if n.AllowUnderscore {
		charset += "_"
	}
	if n.AllowDot {
		charset += "."
	}
	return charset
}

func (r UsernameRules) Validate(username string) (bool, string) {
	n := NormalizeUsernameRules(r)
	length := len(username)
	if length < n.MinLength {
		return false, fmt.Sprintf("username must have at least %d characters", n.MinLength)
	}
	if length > n.MaxLength {
		return false, fmt.Sprintf("username must have at most %d characters", n.MaxLength)
	}

	charset := n.AllowedCharset()
	for i := 0; i < len(username); i++ {
		if strings.IndexByte(charset, username[i]) < 0 {
			return false, "username contains invalid characters"
		}
	}

	if n.DisallowLeadingDot && strings.HasPrefix(username, ".") {
		return false, "username cannot start with dot"
	}

	if n.DisallowTrailingDot && strings.HasSuffix(username, ".") {
		return false, "username cannot end with dot"
	}

	if n.AllowDot && n.MaxConsecutiveDots > 0 {
		currentDots := 0
		for i := 0; i < len(username); i++ {
			if username[i] == '.' {
				currentDots++
				if currentDots > n.MaxConsecutiveDots {
					return false, "username has too many consecutive dots"
				}
			} else {
				currentDots = 0
			}
		}
	}

	return true, "valid"
}
