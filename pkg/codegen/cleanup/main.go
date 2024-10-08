package main

import (
	"os"

	"github.com/rancher/wrangler/v2/pkg/cleanup"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cleanup.Cleanup("./pkg/apis"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./pkg/generated"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./charts/ui-plugin-operator/crds"); err != nil {
		logrus.Fatal(err)
	}
}
