package handler

import (
	"net/http"
	"strings"
)

// Modifier Modifies the outgoing request in place
type Modifier func(req *http.Request)

func HostModifier(req *http.Request) {
	switch req.URL.Path {
	// Yes, BF2Hub really *requires* different host headers for these endpoints
	case "/ASP/getrankstatus.aspx":
		req.Host = "battlefield2.gamestats.gamespy.com"
	case "/ASP/sendsnapshot.aspx":
		req.Host = "gamestats.gamespy.com"
	default:
		req.Host = "BF2Web.gamespy.com"
	}
}

func QueryModifier(req *http.Request) {
	q := req.URL.Query()
	switch req.URL.Path {
	case "/ASP/getplayerinfo.aspx":
		// Use default BFHQ info param to ensure compatibility
		q.Set("info", strings.Join([]string{
			"per*", "cmb*", "twsc", "cpcp", "cacp", "dfcp", "kila", "heal", "rviv", "rsup", "rpar", "tgte", "dkas",
			"dsab", "cdsc", "rank", "cmsc", "kick", "kill", "deth", "suic", "ospm", "klpm", "klpr", "dtpr", "bksk",
			"wdsk", "bbrs", "tcdr", "ban", "dtpm", "lbtl", "osaa", "vrk", "tsql", "tsqm", "tlwf", "mvks", "vmks",
			"mvn*", "vmr*", "fkit", "fmap", "fveh", "fwea", "wtm-", "wkl-", "wdt-", "wac-", "wkd-", "vtm-", "vkl-",
			"vdt-", "vkd-", "vkr-", "atm-", "awn-", "alo-", "abr-", "ktm-", "kkl-", "kdt-", "kkd-",
		}, ","))

	}
	req.URL.RawQuery = q.Encode()
}
