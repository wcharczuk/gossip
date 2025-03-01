package kube

import (
	"bytes"
	"testing"
)

func Test_ToYAML(t *testing.T) {
	d := Deployment("test", "test-image:latest")

	yamlBytes := new(bytes.Buffer)
	err := ToYAML(yamlBytes, d)
	if err != nil {
		t.Fatalf("expected err to be unset, was: %v", err)
	}
	expectedDeployment := "metadata:\n  creationTimestamp: null\n  labels:\n    app: test\n  name: test\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: test\n  strategy:\n    rollingUpdate:\n      maxSurge: 1\n    type: RollingUpdate\n  template:\n    metadata:\n      creationTimestamp: null\n      labels:\n        app: test\n    spec:\n      containers:\n      - image: test-image:latest\n        name: test\n        resources: {}\n      restartPolicy: Always\nstatus: {}\n"
	if yamlBytes.String() != expectedDeployment {
		t.Fatalf("actual yaml contents didn't match expected:\n%s", yamlBytes.String())
	}
}
