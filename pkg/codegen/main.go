package main

import (
	"os"

	v1 "github.com/rancher/ui-plugin-operator/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/ui-plugin-operator/pkg/crd"
	"github.com/sirupsen/logrus"

	controllergen "github.com/rancher/wrangler/v2/pkg/controller-gen"
	"github.com/rancher/wrangler/v2/pkg/controller-gen/args"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "crds" {
		if len(os.Args) != 3 {
			logrus.Fatal("usage: ./codegen crds <crd-directory>")
		}
		logrus.Infof("Writing CRDs to %s", os.Args[2])
		if err := crd.WriteFile(os.Args[2]); err != nil {
			panic(err)
		}
		return
	}

	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/ui-plugin-operator/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"catalog.cattle.io": {
				Types: []interface{}{
					v1.UIPlugin{},
				},
				GenerateTypes: true,
			},
		},
	})
}
