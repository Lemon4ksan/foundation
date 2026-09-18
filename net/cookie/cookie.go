// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cookie implements HTTP State Management parsing, validation, and serialization strictly conforming to RFC 6265 and RFC 6265bis.
package cookie

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// MaxCookieAgeSeconds defines the maximum recommended cookie lifetime in seconds (400 days / 34,560,000s)
// as mandated by RFC 6265bis §5.5.
const (
	MaxCookieAgeSeconds = 34560000
	MaxCookieAgeLimit   = 400 * 24 * time.Hour
)

// Cookie represents an HTTP cookie structure, capturing attributes from Set-Cookie headers
// and formatted for storage and persistence (RFC 6265 §4.1.1, §4.1.2 & §5.3, RFC 6265bis §5.7).
type Cookie struct {
	Expires      time.Time `json:"expires"`
	Name         string    `json:"name"`
	Value        string    `json:"value"`
	Domain       string    `json:"domain"`
	Path         string    `json:"path"`
	SameSite     string    `json:"sameSite,omitempty"`
	PartitionKey string    `json:"partitionKey,omitempty"`
	HTTPOnly     bool      `json:"httpOnly,omitempty"`
	Secure       bool      `json:"secure,omitempty"`
	Partitioned  bool      `json:"partitioned,omitempty"`
	MaxAge       int       `json:"maxAge,omitempty"`
}

// hasProhibitedControlChars reports whether s contains CTL characters %x00-08 / %x0A-1F / %x7F (excluding HTAB %x09)
// per RFC 6265bis §5.5 Step 1 & §5.7 Step 3.
func hasProhibitedControlChars(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if (b <= 0x08) || (b >= 0x0A && b <= 0x1F) || b == 0x7F {
			return true
		}
	}

	return false
}

// ParseSetCookieHeader parses a raw Set-Cookie header line with zero heap allocations (RFC 6265 §5.2, RFC 6265bis §5.5 & §5.7).
func ParseSetCookieHeader(headerVal, defaultDomain, defaultPath string) Cookie {
	if headerVal == "" || hasProhibitedControlChars(headerVal) {
		return Cookie{}
	}

	c := Cookie{
		Domain: defaultDomain,
		Path:   defaultPath,
	}

	isFirst := true
	for key, val := range bytesconv.ScanPairs(headerVal, ';', '=') {
		if isFirst {
			c.Name = key
			c.Value = val
			isFirst = false

			continue
		}

		ParseCookieAttribute(key, val, &c)
	}

	// RFC 6265bis §5.5 Step 5 & §5.7 Step 4: Sum of lengths of name and value must not exceed 4096 octets
	if c.Name == "" || len(c.Name)+len(c.Value) > 4096 {
		return Cookie{}
	}

	return c
}

// ParseCookieAttribute sets the corresponding field on [Cookie] with zero heap allocations using case-insensitive ASCII comparison.
func ParseCookieAttribute(key, val string, c *Cookie) {
	hasVal := len(val) > 0

	// RFC 6265bis §5.5 Step 6: Attribute value longer than 1024 octets must be ignored
	if len(val) > 1024 {
		return
	}

	switch {
	case bytesconv.EqualFoldASCII(key, "httponly"):
		c.HTTPOnly = true
	case bytesconv.EqualFoldASCII(key, "secure"):
		c.Secure = true
	case bytesconv.EqualFoldASCII(key, "partitioned"):
		c.Partitioned = true
	case bytesconv.EqualFoldASCII(key, "samesite"):
		if hasVal {
			switch {
			case bytesconv.EqualFoldASCII(val, "strict"):
				c.SameSite = "Strict"
			case bytesconv.EqualFoldASCII(val, "lax"):
				c.SameSite = "Lax"
			case bytesconv.EqualFoldASCII(val, "none"):
				c.SameSite = "None"
			default:
				c.SameSite = "Default"
			}
		}

	case bytesconv.EqualFoldASCII(key, "domain"):
		if hasVal {
			c.Domain = strings.TrimPrefix(val, ".")
		}
	case bytesconv.EqualFoldASCII(key, "path"):
		if hasVal {
			c.Path = val
		}
	case bytesconv.EqualFoldASCII(key, "max-age"):
		if hasVal {
			if maxAge, err := strconv.Atoi(val); err == nil {
				if maxAge > MaxCookieAgeSeconds {
					maxAge = MaxCookieAgeSeconds
				}

				c.MaxAge = maxAge
			}
		}

	case bytesconv.EqualFoldASCII(key, "expires"):
		if hasVal {
			if exp, err := http.ParseTime(val); err == nil {
				c.Expires = exp
			}
		}
	}
}

// ValidatePrefix verifies whether cookie conforms to RFC 6265bis §4.1.3 & §5.4 cookie prefix rules:
//   - "__Secure-": MUST be set with Secure=true.
//   - "__Host-": MUST be set with Secure=true, Path="/", and empty Domain (host-only).
//   - Nameless cookies whose value begins with "__Secure-" or "__Host-" MUST be rejected (RFC 6265bis §5.7 step 22).
func ValidatePrefix(c Cookie) bool {
	if c.Name == "" {
		lowerVal := strings.ToLower(c.Value)
		if strings.HasPrefix(lowerVal, "__secure-") || strings.HasPrefix(lowerVal, "__host-") {
			return false
		}

		return true
	}

	lowerName := strings.ToLower(c.Name)
	if strings.HasPrefix(lowerName, "__secure-") {
		return c.Secure
	}

	if strings.HasPrefix(lowerName, "__host-") {
		return c.Secure && c.Path == "/" && c.Domain == ""
	}

	return true
}

// PathMatch reports whether reqPath matches cookiePath per RFC 6265 §5.1.4.
func PathMatch(reqPath, cookiePath string) bool {
	if cookiePath == "" {
		cookiePath = "/"
	}

	if reqPath == "" {
		reqPath = "/"
	}

	if reqPath == cookiePath {
		return true
	}

	if strings.HasPrefix(reqPath, cookiePath) {
		if strings.HasSuffix(cookiePath, "/") {
			return true
		}

		if len(reqPath) > len(cookiePath) && reqPath[len(cookiePath)] == '/' {
			return true
		}
	}

	return false
}
