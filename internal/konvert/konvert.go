package konvert

import (
	"os"
	"path"
	"path/filepath"

	"github.com/kumorilabs/konvert/internal/functions"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func Konvert(kpath string) error {
	k, err := New(kpath)
	if err != nil {
		return err
	}
	return k.Run()
}

type Konverter struct {
	path        string
	konvertFile string
}

func (k *Konverter) Run() error {
	fn := &functions.KonvertFunction{}

	// load configuration
	konvertNode, err := kyaml.ReadFile(k.konvertFile)
	if err != nil {
		return err
	}
	if err := fn.Config(konvertNode); err != nil {
		return err
	}

	// run konvert function
	inout := &kio.LocalPackageReadWriter{
		PackagePath: k.path,
	}
	return kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{fn},
		Outputs: []kio.Writer{inout},
	}.Execute()
}

func New(kpath string) (*Konverter, error) {
	konvertFile, path, err := discoverPaths(kpath)
	if err != nil {
		return nil, err
	}

	return &Konverter{path, konvertFile}, nil
}

func discoverPaths(kpath string) (string, string, error) {
	var (
		konvertFile string
		baseDir     string
	)
	finfo, err := os.Stat(kpath)
	if err != nil {
		return konvertFile, baseDir, err
	}
	if finfo.IsDir() {
		// can we find a konvert.yaml?
		baseDir = kpath
		konvertFile = filepath.Join(baseDir, "konvert.yaml")
		_, err = os.Stat(kpath)
		if err != nil {
			return konvertFile, baseDir, err
		}
	} else {
		konvertFile = kpath
		baseDir = path.Dir(konvertFile)
	}

	return konvertFile, baseDir, nil
}
