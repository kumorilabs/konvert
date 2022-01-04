package konvert

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKonverter(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "konverter-new")
	require.NoError(t, err, "TempDir")
	defer func() {
		os.RemoveAll(tmpDir)
	}()
	baseDir := filepath.Join(tmpDir, "base")
	err = os.MkdirAll(baseDir, 0755)
	require.NoError(t, err, "MkdirAll")

	err = ioutil.WriteFile(
		filepath.Join(baseDir, "konvert.yaml"),
		[]byte("\n"),
		0644,
	)
	require.NoError(t, err, "WriteFile")

	wd, err := os.Getwd()
	require.NoError(t, err, "Getwd")
	require.NoError(t, os.Chdir(tmpDir), "Chdir-temp")
	defer func() {
		require.NoError(t, os.Chdir(wd), "Chdir-test")
	}()

	paths := []string{"base/konvert.yaml", "base"}
	for _, path := range paths {
		k, err := New(path)
		require.NoError(t, err, "New")

		assert.Equal(t, "base", k.path, "path")
		assert.Equal(t, "base/konvert.yaml", k.konvertFile, "konvertFile")
	}
}

func TestKonverterRun(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "konverter-run")
	require.NoError(t, err, "TempDir")
	defer func() {
		os.RemoveAll(tmpDir)
	}()
	baseDir := filepath.Join(tmpDir, "base")
	err = os.MkdirAll(baseDir, 0755)
	require.NoError(t, err, "MkdirAll")

	konvertyaml := `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: cluster-autoscaler
spec:
  chart: cluster-autoscaler
  kustomize: true
  repo: https://kubernetes.github.io/autoscaler
  version: 9.11.0
  namespace: cas
`

	err = ioutil.WriteFile(
		filepath.Join(baseDir, "konvert.yaml"),
		[]byte(konvertyaml),
		0644,
	)
	require.NoError(t, err, "WriteFile")

	err = Konvert(baseDir)
	require.NoError(t, err, "Konvert")

	files, err := os.ReadDir(baseDir)
	require.NoError(t, err, "ReadDir")
	assert.True(t, len(files) > 1, "chart-rendered")
}
