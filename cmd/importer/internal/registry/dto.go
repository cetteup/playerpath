package registry

type PlayersResponse struct {
	Players []PlayerDTO `json:"players"`
	HasMore bool        `json:"hasMore"`
}

type PlayerDTO struct {
	ID         string `json:"id"`
	PID        int    `json:"pid"`
	Nick       string `json:"nick"`
	Provider   string `json:"provider"`
	ProfileURL string `json:"profileUrl"`
}
