package konvert

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/kumorilabs/konvert/internal/functions"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// if path points directly to a konvert-file, load it and run
// if it is directory, discover all konvert-files and run against each one

func Konvert(kpath string) error {
	k, err := New(kpath)
	if err != nil {
		return err
	}
	return k.Run()
}

type Konverter struct {
	path string
	fns  []kio.Filter
}

func (k *Konverter) Run() error {
	inout := &kio.LocalPackageReadWriter{
		PackagePath: k.path,
	}
	return kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: k.fns,
		Outputs: []kio.Writer{inout},
	}.Execute()
}

func New(kpath string) (*Konverter, error) {
	var (
		konvertfns []kio.Filter
		basedir    string
	)

	log.WithField("path", kpath).Debug("creating Konverter")
	finfo, err := os.Stat(kpath)
	if err != nil {
		return nil, err
	}

	if finfo.IsDir() {
		basedir = kpath
		fns, err := discoverFns(kpath)
		if err != nil {
			return nil, err
		}
		konvertfns = fns
	} else {
		basedir = path.Dir(kpath)
		fn, err := loadFn(kpath)
		if err != nil {
			return nil, err
		}
		konvertfns = []kio.Filter{fn}
	}

	return &Konverter{basedir, konvertfns}, nil
}

func loadFn(kpath string) (kio.Filter, error) {
	konvertNode, err := kyaml.ReadFile(kpath)
	if err != nil {
		return nil, err
	}
	log.WithField("path", kpath).Debug("adding Konvert fn")
	fn := functions.Konvert(kpath)
	if err := fn.Config(konvertNode); err != nil {
		return nil, err
	}
	return fn, nil
}

func discoverFns(pkgpath string) ([]kio.Filter, error) {
	var konvertfns []kio.Filter
	reader := kio.LocalPackageReader{PackagePath: pkgpath}
	rnodes, err := reader.Read()
	if err != nil {
		return konvertfns, err
	}

	for _, rnode := range rnodes {
		if functions.IsKonvertFile(rnode) {
			path, _, err := kioutil.GetFileAnnotations(rnode)
			if err != nil {
				return konvertfns, fmt.Errorf("getting file annotations: %w", err)
			}
			path = filepath.Join(pkgpath, path)

			log.WithField("path", path).Debug("adding Konvert fn")
			fn := functions.Konvert(path)
			if err := fn.Config(rnode); err != nil {
				return konvertfns, err
			}
			konvertfns = append(konvertfns, fn)
		}
	}

	return konvertfns, err
}
