package main

import (
	"log"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func main() {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "memory", log.Printf); err != nil {
		log.Fatalf("Error initializing action configuration: %v", err)
	}

	pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
	pull.RepoURL = "https://charts.bitnami.com/bitnami"
	pull.DestDir = filepath.Join(".")
	pull.Untar = false // Cambiar a true si se desea descomprimir el chart

	chartName := "nginx"
	if _, err := pull.Run(chartName); err != nil {
		log.Fatalf("Error pulling chart: %v", err)
	}
}
