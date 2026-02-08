package modify

import (
	"net/http"

	"github.com/cetteup/playerpath/internal/domain/provider"
)

type ModifierType int

const (
	ModifierTypeRequest  ModifierType = iota
	ModifierTypeResponse ModifierType = iota
)

type Modifier interface {
	Type() ModifierType
}

type RequestModifier interface {
	Modifier
	Modify(pv provider.Provider, req *http.Request) error
}

type ResponseModifier interface {
	Modifier
	Modify(pv provider.Provider, res *http.Response) error
}
