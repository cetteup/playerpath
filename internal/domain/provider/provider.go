package provider

import (
	"fmt"
)

type Provider int

const (
	ProviderUnknown Provider = 0
	ProviderBF2Hub  Provider = 1
	ProviderPlayBF2 Provider = 2
	ProviderOpenSpy Provider = 3
	ProviderB2BF2   Provider = 4

	providerNameBF2Hub  = "bf2hub"
	providerNamePlayBF2 = "playbf2"
	providerNameOpenSpy = "openspy"
	providerNameB2BF2   = "b2bf2"

	baseURLBF2Hub  = "http://official.ranking.bf2hub.com/"
	baseURLPlayBF2 = "http://bf2web.playbf2.ru/"
	baseURLOpenSpy = "http://bf2web.openspy.net/"
	baseURLB2BF2   = "https://stats.b2bf2.net/"
)

//goland:noinspection GoMixedReceiverTypes
func (p Provider) String() string {
	switch p {
	case ProviderBF2Hub:
		return providerNameBF2Hub
	case ProviderPlayBF2:
		return providerNamePlayBF2
	case ProviderOpenSpy:
		return providerNameOpenSpy
	case ProviderB2BF2:
		return providerNameB2BF2
	default:
		return "unknown"
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) BaseURL() string {
	switch p {
	case ProviderBF2Hub:
		return baseURLBF2Hub
	case ProviderPlayBF2:
		return baseURLPlayBF2
	case ProviderOpenSpy:
		return baseURLOpenSpy
	case ProviderB2BF2:
		return baseURLB2BF2
	default:
		return "http://unknown"
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) RequiresGameSpyHost() bool {
	switch p {
	// BF2Hub only serves ASP requests with original gamespy.com host headers
	case ProviderBF2Hub:
		return true
	default:
		return false
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) RequiresBFHQInfoQuery() bool {
	switch p {
	// BF2Hub only returns player info if the info query parameter matches the one used for the in-game BFHQ
	case ProviderBF2Hub:
		return true
	default:
		return false
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) SupportsPlayerVerification() bool {
	switch p {
	case ProviderPlayBF2, ProviderOpenSpy, ProviderB2BF2:
		return true
	default:
		return false
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) AllowsCaseInsensitiveLogin() bool {
	switch p {
	// BF2Hub allows players to log in with any spelling/casing of their name
	case ProviderBF2Hub:
		return true
	default:
		return false
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p *Provider) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = ProviderUnknown
		return nil
	}

	s := string(text)
	switch s {
	case providerNameBF2Hub:
		*p = ProviderBF2Hub
	case providerNamePlayBF2:
		*p = ProviderPlayBF2
	case providerNameOpenSpy:
		*p = ProviderOpenSpy
	case providerNameB2BF2:
		*p = ProviderB2BF2
	default:
		return fmt.Errorf("invalid provider: %s", s)
	}

	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (p Provider) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}
