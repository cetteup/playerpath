package registry

type Player struct {
	ID         string `json:"id"`
	PID        int    `json:"pid"`
	Nick       string `json:"nick"`
	Provider   string `json:"provider"`
	ProfileURL string `json:"profileUrl"`
}
