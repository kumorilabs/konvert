package konverter

import (
	"github.com/kumorilabs/konvert/pkg/sources"
)

type Transformer interface {
	Transform([]sources.Resource) ([]sources.Resource, error)
	Name() string
}
