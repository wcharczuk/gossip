package clifactory

import (
	"flag"
	"gossip/pkg/kube"
	"os"
)

type Resources map[string]any

func (r Resources) Main() {
	flagName := flag.String("name", "", "The resource name")
	flag.Parse()
	if resource, ok := r[*flagName]; ok {
		_ = kube.ToYAML(os.Stdout, resource)
	}
}
