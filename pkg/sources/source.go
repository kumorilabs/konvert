package sources

// Source defines an application source
type Source interface {
	Fetch() error
	Generate() error
	Kustomize() error
}
