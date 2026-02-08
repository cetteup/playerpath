package modify

import (
	"net/http"
	"strings"

	"github.com/cetteup/playerpath/internal/domain/provider"
)

type HostRequestModifier struct{}

func (m HostRequestModifier) Type() ModifierType {
	return ModifierTypeRequest
}

func (m HostRequestModifier) Modify(pv provider.Provider, req *http.Request) error {
	if m.skip(pv, req) {
		return nil
	}

	// Yes, BF2Hub really *requires* different host headers for some endpoints
	switch req.URL.Path {
	case "/ASP/getrankstatus.aspx":
		req.Host = "battlefield2.gamestats.gamespy.com"
	case "/ASP/sendsnapshot.aspx":
		req.Host = "gamestats.gamespy.com"
	default:
		req.Host = "BF2Web.gamespy.com"
	}

	return nil
}

func (m HostRequestModifier) skip(pv provider.Provider, _ *http.Request) bool {
	return !provider.RequiresGameSpyHost(pv)
}

type InfoQueryRequestModifier struct{}

func (m InfoQueryRequestModifier) Type() ModifierType {
	return ModifierTypeRequest
}

func (m InfoQueryRequestModifier) Modify(pv provider.Provider, req *http.Request) error {
	if m.skip(pv, req) {
		return nil
	}

	q := req.URL.Query()
	// Use default BFHQ info param to ensure compatibility
	q.Set("info", strings.Join([]string{
		"per*", "cmb*", "twsc", "cpcp", "cacp", "dfcp", "kila", "heal", "rviv", "rsup", "rpar", "tgte", "dkas", "dsab",
		"cdsc", "rank", "cmsc", "kick", "kill", "deth", "suic", "ospm", "klpm", "klpr", "dtpr", "bksk", "wdsk", "bbrs",
		"tcdr", "ban", "dtpm", "lbtl", "osaa", "vrk", "tsql", "tsqm", "tlwf", "mvks", "vmks", "mvn*", "vmr*", "fkit",
		"fmap", "fveh", "fwea", "wtm-", "wkl-", "wdt-", "wac-", "wkd-", "vtm-", "vkl-", "vdt-", "vkd-", "vkr-", "atm-",
		"awn-", "alo-", "abr-", "ktm-", "kkl-", "kdt-", "kkd-",
	}, ","))
	req.URL.RawQuery = q.Encode()

	return nil
}

func (m InfoQueryRequestModifier) skip(pv provider.Provider, req *http.Request) bool {
	return !provider.RequiresBFHQInfoQuery(pv) || req.URL.Path != "/ASP/getplayerinfo.aspx"
}
