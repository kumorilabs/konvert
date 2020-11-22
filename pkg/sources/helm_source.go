package sources

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/equality"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// helmSource represents a helm chart
type helmSource struct {
	Repo            string                 `json:"repo,omitempty" yaml:"repo,omitempty"`
	RepoSubPath     string                 `json:"repoSubPath,omitempty" yaml:"repoSubPath,omitempty"`
	ChartName       string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Version         string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Values          map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	NamespaceConfig namespaceConfig        `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	localChart      bool
	log             *log.Entry
}

type namespaceConfig struct {
	Name   string            `json:"name,omitempty" yaml:"name,omitempty"`
	Create bool              `json:"create,omitempty" yaml:"create,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// NewHelmSourceFromConfig creates a new Helm Source from a config map
func NewHelmSourceFromConfig(config map[string]interface{}) (Source, error) {
	var source *helmSource

	// marshal map into a byte slice
	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling helm source config")
	}

	// unmarshal into a helmSource
	err = yaml.Unmarshal(b, &source)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling helm source config")
	}

	if strings.Contains(source.ChartName, "/") {
		nameParts := strings.Split(source.ChartName, "/")
		log.Debug("nameParts length is: ", len(nameParts))
		if len(nameParts) > 1 {
			source.Repo = nameParts[0]
			source.ChartName = nameParts[len(nameParts)-1]

			if len(nameParts) > 2 {
				repoSubPath := strings.Join(nameParts[1:len(nameParts)-1], "/")

				source.RepoSubPath = repoSubPath
			}
		}
	}

	log.Debug("ChartName is: ", source.ChartName)
	log.Debug("Repo is: ", source.Repo)
	log.Debug("RepoSubPath is: ", source.RepoSubPath)

	if source.Repo == "." || source.Repo == "" {
		source.localChart = true
		log.Debug("Local chart detected")
	}

	source.log = log.WithFields(log.Fields{
		"pkg":    "sources",
		"source": "helm",
	})

	return source, nil
}

// Fetch downloads the source
func (h *helmSource) Fetch() error {
	if h.localChart {
		h.log.Debug("using local chart, skipping fetch")
		return nil
	}

	chartYaml := filepath.Join(h.chartDir(), h.Name(), "Chart.yaml")
	if _, err := os.Stat(chartYaml); err == nil {
		// chart is already extracted, return
		h.log.Debug("found chart in cache")
		return nil
	}

	cachedir := cacheDir()
	if err := os.MkdirAll(cachedir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "error creating cache dir %s", cachedir)
	}

	cmd := h.fetchCommand()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "error running fetch command: %q", string(out))
	}

	return nil
}

// Generate creates raw manifests
func (h *helmSource) Generate() ([]Resource, error) {
	var valuesFile string

	h.log.Debugf("Values are is: %s", h.Values)

	if len(h.Values) > 0 {
		tmpfile, err := ioutil.TempFile("", fmt.Sprintf("konvert-%s-*", h.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "error creating values file")
		}

		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		b, err := yaml.Marshal(h.Values)
		if err != nil {
			return nil, errors.Wrap(err, "error marshaling values")
		}

		if _, err := tmpfile.Write(b); err != nil {
			return nil, errors.Wrap(err, "error writing values file")
		}

		valuesFile = tmpfile.Name()
	}

	h.log.Debugf("Values file is: %s", valuesFile)

	cmd := h.templateCommand(valuesFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.Bytes()
	outErr := stderr.Bytes()

	h.log.Debugf("template command: Stdout:%s", out)
	h.log.Debugf("template command: Stderr:%s", outErr)

	if err != nil {
		return nil, errors.Wrapf(err, "error running template command: %s\n", outErr)
	}

	if len(outErr) > 0 {
		h.log.Infof("template command error output: %s", outErr)
	}

	resources, err := h.decode(bytes.NewReader(out))
	if err != nil {
		return resources, errors.Wrap(err, "error decoding template output")
	}

	h.log.Debugf("found %d resources", len(resources))

	crdsDir := filepath.Join(h.chartDirRoot(), "crds")
	if _, err := os.Stat(crdsDir); !os.IsNotExist(err) {

		files, err := ioutil.ReadDir(crdsDir)
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding crds: %s\n", err)
		}

		for _, file := range files {
			out, err := ioutil.ReadFile(filepath.Join(crdsDir, file.Name()))
			if err != nil {
				return nil, errors.Wrapf(err, "error decoding crds: %s\n", err)
			}
			crds, err := h.decode(bytes.NewReader(out))
			if err != nil {
				return crds, errors.Wrap(err, "error decoding crds")
			}
			for _, crd := range crds {
				if !h.containsResource(resources, crd) {
					resources = append(resources, crd)
				}
			}
		}
	}

	return resources, nil
}

func (h *helmSource) containsResource(resources []Resource, resource Resource) bool {

	for _, r := range resources {
		if equality.Semantic.DeepEqual(r, resource) {
			return true
		}
	}
	return false
}

// Name returns the name of this source
func (h *helmSource) Name() string {
	return h.ChartName
}

// Namespace returns the namespace this chart will be installed in
func (h *helmSource) Namespace() string {
	ns := h.NamespaceConfig.Name
	if ns == "" {
		ns = h.Name()
	}
	return ns
}

// CreateNamespace returns true if the source include a Namespace resource
func (h *helmSource) CreateNamespace() bool {
	return h.NamespaceConfig.Create
}

// NamespaceLabels returns the labels that should be set on the namespace (if create)
func (h *helmSource) NamespaceLabels() map[string]string {
	return h.NamespaceConfig.Labels
}

func (h *helmSource) templateCommand(valuesFile string) *exec.Cmd {
	args := []string{
		"template",
		h.Name(),
		"--namespace", h.Namespace(),
	}

	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}

	args = append(args, filepath.Join(h.chartDirRoot()))

	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm template command")

	return cmd
}

func (h *helmSource) fetchCommand() *exec.Cmd {
	args := []string{"fetch", "--untar"}
	if h.Version != "" {
		args = append(args, "--version", h.Version)
	}

	repo := h.Repo
	h.log.Debugf("Repo path is : %s", repo)

	h.log.Debugf("RepoSubPath is : %s", h.RepoSubPath)
	if h.RepoSubPath != "" {

		repo = repo + "/" + h.RepoSubPath
	}

	args = append(
		args,
		"--destination", h.chartDir(),
		repo+"/"+h.Name(),
	)
	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm fetch command")

	return cmd
}

func (h *helmSource) decode(in io.Reader) ([]Resource, error) {
	var (
		result []Resource
		err    error
	)

	decoder := k8syaml.NewYAMLOrJSONDecoder(in, 1024)

	for err == nil {
		var out Resource
		err = decoder.Decode(&out)
		if err == nil && len(out.Object) > 0 {
			if out.IsList() {
				items, err := out.ToList()
				if err != nil {
					return nil, errors.Wrap(err, "failed to explode list")
				}
				result = append(result, items...)
			} else {
				result = append(result, out)
			}
		}
	}
	if err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode input")
	}

	return result, nil
}

func (h *helmSource) chartDirRoot() string {
	if h.localChart {
		return filepath.Join(h.chartDir(), h.RepoSubPath, h.Name())
	}

	return filepath.Join(h.chartDir(), h.Name())

}

func (h *helmSource) chartDir() string {
	if h.localChart {
		return "."
	}
	var d string
	if h.Version != "" {
		d = fmt.Sprintf("%s-%s", h.Name(), h.Version)
	} else {
		d = h.Name()
	}

	return filepath.Join(
		cacheDir(),
		h.Repo,
		d,
	)
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helm")
}
