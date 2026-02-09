//go:generate go tool stringer -type=Provider
package provider

import (
	"fmt"
	"strings"
)

type Provider int

const (
	Unknown Provider = 0
	BF2Hub  Provider = 1
	PlayBF2 Provider = 2
	OpenSpy Provider = 3
	B2BF2   Provider = 4

	baseURLBF2Hub  = "http://official.ranking.bf2hub.com/"
	baseURLPlayBF2 = "http://bf2web.playbf2.ru/"
	baseURLOpenSpy = "http://bf2web.openspy.net/"
	baseURLB2BF2   = "https://stats.b2bf2.net/"
)

//goland:noinspection GoMixedReceiverTypes
func (p *Provider) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = Unknown
		return nil
	}

	s := string(text)
	if strings.EqualFold(s, BF2Hub.String()) {
		*p = BF2Hub
	} else if strings.EqualFold(s, PlayBF2.String()) {
		*p = PlayBF2
	} else if strings.EqualFold(s, OpenSpy.String()) {
		*p = OpenSpy
	} else if strings.EqualFold(s, B2BF2.String()) {
		*p = B2BF2
	} else {
		return fmt.Errorf("invalid provider: %s", s)
	}

	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}

func GetBaseURL(p Provider) string {
	switch p {
	case BF2Hub:
		return baseURLBF2Hub
	case PlayBF2:
		return baseURLPlayBF2
	case OpenSpy:
		return baseURLOpenSpy
	case B2BF2:
		return baseURLB2BF2
	default:
		return "http://unknown"
	}
}

func RequiresGameSpyHost(p Provider) bool {
	switch p {
	// BF2Hub only serves ASP requests with original gamespy.com host headers
	case BF2Hub:
		return true
	default:
		return false
	}
}

func RequiresBFHQInfoQuery(p Provider) bool {
	switch p {
	// BF2Hub only returns player info if the info query parameter matches the one used for the in-game BFHQ
	case BF2Hub:
		return true
	default:
		return false
	}
}

func SupportsStandardPlayerVerification(p Provider) bool {
	switch p {
	case PlayBF2, OpenSpy, B2BF2:
		return true
	default:
		return false
	}
}

func AllowsCaseInsensitiveLogin(p Provider) bool {
	switch p {
	// BF2Hub allows players to log in with any spelling/casing of their name
	case BF2Hub:
		return true
	default:
		return false
	}
}
