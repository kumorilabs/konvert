package konverter

import (
	"github.com/ryane/konvert/pkg/sources"
)

type Transformer interface {
	Transform([]sources.Resource) ([]sources.Resource, error)
	Name() string
}
