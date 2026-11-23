package security

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"backend_si/internal/domain"
)

// Post limits (005 D-4), counted in Unicode code points after trimming.
// The posts table repeats them as CHECK constraints.
const (
	MaxPostTitleLength = 120
	MaxPostBodyLength  = 2000
)

// MaxRemovalReasonLength is the longest moderation reason (006 D-2), in
// code points after trimming. The posts table repeats it as a CHECK.
const MaxRemovalReasonLength = 500

// MaxPostPage is the highest page of the wall a client may ask for (D-5).
const MaxPostPage = 10000

// pagePattern and idPattern accept plain decimal digits only: no sign, no
// spaces, no leading zero.
var (
	pagePattern = regexp.MustCompile(`^[1-9]\d{0,4}$`)
	idPattern   = regexp.MustCompile(`^[1-9]\d{0,18}$`)
)

// ValidatePost trims the title and the body and checks them (D-4): valid
// UTF-8, 1 to MaxPostTitleLength code points of title with no control
// characters, and 1 to MaxPostBodyLength code points of body where the only
// control characters allowed are \n, \r and \t. It returns the trimmed
// values. Failures wrap domain.ErrInvalidInput.
func (s *Security) ValidatePost(title, body string) (string, string, error) {
	title, err := postField("title", title, MaxPostTitleLength, func(rune) bool { return false })
	if err != nil {
		return "", "", err
	}
	body, err = postField("body", body, MaxPostBodyLength, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t'
	})
	if err != nil {
		return "", "", err
	}
	return title, body, nil
}

// ValidateRemovalReason trims the reason of a moderation removal and checks
// it like a post body (006 D-2): valid UTF-8, 1 to MaxRemovalReasonLength
// code points, and no control characters other than \n, \r and \t. It
// returns the trimmed reason. Failures wrap domain.ErrInvalidInput.
func (s *Security) ValidateRemovalReason(reason string) (string, error) {
	return postField("removal reason", reason, MaxRemovalReasonLength, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t'
	})
}

func postField(name, value string, maxLen int, allowedControl func(rune) bool) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("security: validate post %s: invalid utf-8: %w", name, domain.ErrInvalidInput)
	}
	value = strings.TrimSpace(value)
	if n := utf8.RuneCountInString(value); n < 1 || n > maxLen {
		return "", fmt.Errorf("security: validate post %s: length: %w", name, domain.ErrInvalidInput)
	}
	for _, r := range value {
		if unicode.IsControl(r) && !allowedControl(r) {
			return "", fmt.Errorf("security: validate post %s: control character: %w", name, domain.ErrInvalidInput)
		}
	}
	return value, nil
}

// ParsePage reads the page query parameter of the wall (D-5): empty is page
// 1, otherwise an integer from 1 to MaxPostPage. Anything else wraps
// domain.ErrInvalidInput (400).
func (s *Security) ParsePage(raw string) (int, error) {
	if raw == "" {
		return 1, nil
	}
	if !pagePattern.MatchString(raw) {
		return 0, fmt.Errorf("security: parse page: %w", domain.ErrInvalidInput)
	}
	page, err := strconv.Atoi(raw)
	if err != nil || page > MaxPostPage {
		return 0, fmt.Errorf("security: parse page: %w", domain.ErrInvalidInput)
	}
	return page, nil
}

// ParsePostID reads a post ID from the path. A malformed ID wraps
// domain.ErrNotFound, so it gets the same 404 as a missing post or another
// user's post (D-3).
func (s *Security) ParsePostID(raw string) (domain.PostID, error) {
	if !idPattern.MatchString(raw) {
		return 0, fmt.Errorf("security: parse post id: %w", domain.ErrNotFound)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("security: parse post id: %w", domain.ErrNotFound)
	}
	return domain.PostID(id), nil
}
