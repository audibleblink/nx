package main

import (
	"embed"

	"github.com/audibleblink/logerr"

	"github.com/audibleblink/nx/internal/cmd"
)

// Embed the plugins directory
//
//go:embed plugins/*
var bundledPlugins embed.FS

func init() {
	// Set up logging
	logerr.SetContextSeparator(" ❯ ")
	logerr.SetLogLevel(logerr.LogLevelInfo)
	logerr.EnableColors()
	logerr.EnableTimestamps()
	logerr.SetContext("nx")
}

func main() {
	cmd.Execute(bundledPlugins)
}
