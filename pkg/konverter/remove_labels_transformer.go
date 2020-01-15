package konverter

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/ryane/konvert/pkg/sources"
	log "github.com/sirupsen/logrus"
)

type RemoveLabelsTransformerConfig struct {
	Names []string `json:"names"`
}

type RemoveLabelsTransformer struct {
	names []string
	log   *log.Entry
}

func NewRemoveLabelsTransformer(c RemoveLabelsTransformerConfig) *RemoveLabelsTransformer {
	return &RemoveLabelsTransformer{
		names: c.Names,
		log: log.WithFields(log.Fields{
			"pkg":         "konverter",
			"transformer": "RemoveLabelsTransformer",
		}),
	}
}

func (t *RemoveLabelsTransformer) Name() string { return "RemoveLabelsTransformer" }

func (t *RemoveLabelsTransformer) Transform(resources []sources.Resource) ([]sources.Resource, error) {
	for _, res := range resources {
		for _, lp := range labelPaths {
			if !lp.IsMatch(res) {
				continue
			}
			log.Infof("Check %s in %s", lp.path, res.GroupVersionKind().String())

			if err := t.removeLabelsAtPath(lp.Fields(), res.UnstructuredContent()); err != nil {
				return resources, errors.Wrapf(
					err,
					"error removing labels for %s",
					res.GroupVersionKind().String(),
				)
			}
		}
	}
	return resources, nil
}

func (t *RemoveLabelsTransformer) removeLabelsAtPath(fieldPath []string, m map[string]interface{}) error {
	var (
		rawval interface{}
		ok     bool
	)

	if len(fieldPath) == 0 {
		return nil
	}

	firstField := fieldPath[0]
	log.Debugf("Checking field %s", firstField)
	if rawval, ok = m[firstField]; !ok {
		log.Warnf("%s not found. next", firstField)
		return nil
	}

	if len(fieldPath) == 1 {
		lblmap, ok := rawval.(map[string]interface{})
		if ok && lblmap != nil {
			for _, k := range t.names {
				if l, found := lblmap[k]; found && l != nil {
					log.Infof("removing %q from %+v", k, lblmap)
					delete(lblmap, k)
				}
			}
		}
	}

	newPath := fieldPath[1:]
	switch valtype := rawval.(type) {
	case nil:
		log.Debugf("no labels to remove at %s", firstField)
	case map[string]interface{}:
		return t.removeLabelsAtPath(newPath, valtype)
	case []interface{}:
		for _, ty := range valtype {
			err := t.removeLabelsAtPath(newPath, ty.(map[string]interface{}))
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected type %T: %#v", valtype, valtype)
	}

	return nil
}

type labelPath struct {
	path    string
	group   string
	version string
	kind    string
}

func (p *labelPath) Fields() []string {
	return strings.Split(p.path, "/")
}

func (p *labelPath) IsMatch(res sources.Resource) bool {
	gvk := res.GroupVersionKind()
	if len(p.group) > 0 && gvk.Group != p.group {
		return false
	}
	if len(p.version) > 0 && gvk.Version != p.version {
		return false
	}
	if len(p.kind) > 0 && gvk.Kind != p.kind {
		return false
	}
	return true
}

var labelPaths = []labelPath{
	{
		path: "metadata/labels",
	},
	{
		path:    "spec/selector",
		version: "v1",
		kind:    "Service",
	},
	{
		path:    "spec/selector",
		version: "v1",
		kind:    "ReplicationController",
	},
	{
		path:    "spec/template/metadata/labels",
		version: "v1",
		kind:    "ReplicationController",
	},
	{
		path: "spec/selector/matchLabels",
		kind: "Deployment",
	},
	{
		path: "spec/template/metadata/labels",
		kind: "Deployment",
	},
	{
		path:  "spec/template/spec/affinity/podAffinity/preferredDuringSchedulingIgnoredDuringExecution/podAffinityTerm/labelSelector/matchLabels",
		group: "apps",
		kind:  "Deployment",
	},
	{
		path:  "spec/template/spec/affinity/podAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels",
		group: "apps",
		kind:  "Deployment",
	},
	{
		path:  "spec/template/spec/affinity/podAntiAffinity/preferredDuringSchedulingIgnoredDuringExecution/podAffinityTerm/labelSelector/matchLabels",
		group: "apps",
		kind:  "Deployment",
	},
	{
		path:  "spec/template/spec/affinity/podAntiAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels",
		group: "apps",
		kind:  "Deployment",
	},
	{
		path: "spec/selector/matchLabels",
		kind: "ReplicaSet",
	},
	{
		path: "spec/template/metadata/labels",
		kind: "ReplicaSet",
	},
	{
		path: "spec/selector/matchLabels",
		kind: "DaemonSet",
	},
	{
		path: "spec/template/metadata/labels",
		kind: "DaemonSet",
	},
	{
		path:  "spec/selector/matchLabels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/template/metadata/labels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/template/spec/affinity/podAffinity/preferredDuringSchedulingIgnoredDuringExecution/podAffinityTerm/labelSelector/matchLabels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/template/spec/affinity/podAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/template/spec/affinity/podAntiAffinity/preferredDuringSchedulingIgnoredDuringExecution/podAffinityTerm/labelSelector/matchLabels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/template/spec/affinity/podAntiAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/volumeClaimTemplates[]/metadata/labels",
		group: "apps",
		kind:  "StatefulSet",
	},
	{
		path:  "spec/selector/matchLabels",
		group: "batch",
		kind:  "Job",
	},
	{
		path:  "spec/template/metadata/labels",
		group: "batch",
		kind:  "Job",
	},
	{
		path:  "spec/jobTemplate/spec/selector/matchLabels",
		group: "batch",
		kind:  "CronJob",
	},
	{
		path:  "spec/jobTemplate/metadata/labels",
		group: "batch",
		kind:  "CronJob",
	},
	{
		path:  "spec/jobTemplate/spec/template/metadata/labels",
		group: "batch",
		kind:  "CronJob",
	},
	{
		path:  "spec/selector/matchLabels",
		group: "policy",
		kind:  "PodDisruptionBudget",
	},
	{
		path:  "spec/podSelector/matchLabels",
		group: "networking.k8s.io",
		kind:  "NetworkPolicy",
	},
	{
		path:  "spec/ingress/from/podSelector/matchLabels",
		group: "networking.k8s.io",
		kind:  "NetworkPolicy",
	},
	{
		path:  "spec/egress/to/podSelector/matchLabels",
		group: "networking.k8s.io",
		kind:  "NetworkPolicy",
	},
}
