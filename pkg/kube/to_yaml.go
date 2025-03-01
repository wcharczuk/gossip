package kube

import (
	"encoding/json"
	"io"

	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

// ToYAML emits a given object as yaml.
//
// In practice, this will wash the object through the JSON representation first.
func ToYAML(wr io.Writer, obj any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var jsonRawRoot any
	if err := json.Unmarshal(jsonBytes, &jsonRawRoot); err != nil {
		return err
	}

	return yaml.NewEncoder(wr).Encode(jsonRawRoot)
}
