package konvert

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKonverter(t *testing.T) {
	var tests = []struct {
		name            string
		path            string
		konvertFiles    []string
		expectedFnCount int
	}{
		{
			name: "single-file-path",
			path: "konvert.yaml",
			konvertFiles: []string{
				"konvert.yaml",
			},
			expectedFnCount: 1,
		},
		{
			name: "single-file-path-with-multiple-in-package",
			path: "konvert.yaml",
			konvertFiles: []string{
				"konvert.yaml",
				"another-konvert.yaml",
			},
			expectedFnCount: 1,
		},
		{
			name: "dir-path",
			path: "",
			konvertFiles: []string{
				"konvert.yaml",
			},
			expectedFnCount: 1,
		},
		{
			name: "dir-path-with-multiple-in-package",
			path: "",
			konvertFiles: []string{
				"konvert.yaml",
				"another-konvert.yaml",
				"nested/konvert.yaml",
			},
			expectedFnCount: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "konverter-new")
			require.NoError(t, err, "TempDir")
			defer func() {
				os.RemoveAll(tmpDir)
			}()
			baseDir := filepath.Join(tmpDir, "base")
			err = os.MkdirAll(baseDir, 0755)
			require.NoError(t, err, "MkdirAll")

			for _, kf := range test.konvertFiles {
				err = testWriteKonvert(baseDir, kf)
				require.NoError(t, err, "testWriteKonvert")
			}

			path := filepath.Join(baseDir, test.path)

			k, err := New(path)
			require.NoError(t, err, "New")
			assert.Equal(t, baseDir, k.path, "path")
			assert.Equal(t, test.expectedFnCount, len(k.fns), "fns")
		})
	}
}

func TestKonverterRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "konverter-run")
	require.NoError(t, err, "TempDir")
	defer func() {
		os.RemoveAll(tmpDir)
	}()
	baseDir := filepath.Join(tmpDir, "base")
	err = os.MkdirAll(baseDir, 0755)
	require.NoError(t, err, "MkdirAll")

	err = testWriteKonvertYAML(
		baseDir,
		"konvert.yaml",
		"cluster-autoscaler",
		"https://kubernetes.github.io/autoscaler",
		"9.11.0",
		"cas",
	)
	require.NoError(t, err, "testWriteKonvertYAML")

	err = Konvert(baseDir)
	require.NoError(t, err, "Konvert")

	files, err := os.ReadDir(baseDir)
	require.NoError(t, err, "ReadDir")
	assert.True(t, len(files) > 1, "chart-rendered")
}

func testWriteKonvert(baseDir, filename string) error {
	return testWriteKonvertYAML(
		baseDir,
		filename,
		"cluster-autoscaler",
		"https://kubernetes.github.io/autoscaler",
		"9.11.0",
		"cas",
	)
}

func testWriteKonvertYAML(baseDir, filename, chart, repo, version, namespace string) error {
	konvertyaml := fmt.Sprintf(`apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: %s
spec:
  chart: %s
  kustomize: true
  repo: %s
  version: %s
  namespace: %s
`,
		chart, chart, repo, version, namespace,
	)

	fn := filepath.Join(baseDir, filename)
	if err := os.MkdirAll(filepath.Dir(fn), 0755); err != nil {
		return err
	}

	return os.WriteFile(
		fn,
		[]byte(konvertyaml),
		0644,
	)
}
