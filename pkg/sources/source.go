package sources

// Source defines an application source
type Source interface {
	Fetch() error
	Generate() ([]Resource, error)
	Name() string
	Namespace() string
	NamespaceLabels() map[string]string
	CreateNamespace() bool
}
