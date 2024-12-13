package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func bailIfErr(err error) {
	if err != nil {
		slog.Error("Error", "err", err)
		os.Exit(1)
	}
}

func handler(cmd *cobra.Command, method, path string) {
	slog.Info("called!", "method", method, "path", path)
}

func main() {
	model, err := LoadFileV3("api.yaml")
	bailIfErr(err)

	rootCmd := cobra.Command{
		Use:   "bctl",
		Short: "Bob CLI",
		Long:  "This is Bob's CLI mate!",
	}
	handlers := map[string]Handler{
		"ArtifactStoreCreate":    handler,
		"ArtifactStoreDelete":    handler,
		"ArtifactStoreList":      handler,
		"CCTray":                 handler,
		"ClusterInfo":            handler,
		"GetApiSpec":             handler,
		"GetError":               handler,
		"GetEvents":              handler,
		"GetMetrics":             handler,
		"HealthCheck":            handler,
		"PipelineArtifactFetch":  handler,
		"PipelineCreate":         handler,
		"PipelineDelete":         handler,
		"PipelineList":           handler,
		"PipelineLogs":           handler,
		"PipelinePause":          handler,
		"PipelineRuns":           handler,
		"PipelineStart":          handler,
		"PipelineStatus":         handler,
		"PipelineStop":           handler,
		"PipelineUnpause":        handler,
		"Query":                  handler,
		"ResourceProviderCreate": handler,
		"ResourceProviderDelete": handler,
		"ResourceProviderList":   handler,
	}

	bailIfErr(GenCliV3(&rootCmd, *model, handlers))

	rootCmd.Execute()
}
