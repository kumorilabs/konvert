package kube

import (
	"encoding/json"
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func TryDiscoverKubeVersion() (string, error) {
	return tryClientVersion()
}

type versionInfo struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"clientVersion"`
}

func tryClientVersion() (string, error) {
	notFound := ""
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return notFound, errors.Wrap(err, "unable to find kubectl in the path")
	}

	log.WithField("kubectl", kubectl).Debug("getting client version via kubectl")

	cmd := exec.Command("kubectl", "version", "--client=true", "--output=json")
	output, err := cmd.Output()
	if err != nil {
		return notFound, errors.Wrap(err, "unable to exec kubectl version")
	}

	var verinfo versionInfo
	err = json.Unmarshal(output, &verinfo)
	if err != nil {
		return notFound, errors.Wrap(err, "unable to unmarshal kubectl version output")
	}

	return verinfo.ClientVersion.GitVersion, nil
}
