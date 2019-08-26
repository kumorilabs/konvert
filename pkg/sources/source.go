package sources

// Source defines an application source
type Source interface {
	Fetch() error
	Generate() ([]Resource, error)
	Name() string
}
