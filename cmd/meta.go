package cmd

import (
	"fmt"
	"strings"

	"github.com/avinor/tau/pkg/helpers/ui"
	"github.com/avinor/tau/pkg/helpers/paths"
	"github.com/avinor/tau/pkg/terraform"
	"github.com/spf13/cobra"
)

type meta struct {
	timeout int
	file    string

	Engine *terraform.Engine

	TempDir string
}

func (m *meta) processArgs(args []string) error {
	if workingDir == "" {
		workingDir = paths.WorkingDir
	}

	m.TempDir = paths.TempDir(workingDir, m.file)

	ui.Debug("temp dir: %s", m.TempDir)

	{
		m.Engine = terraform.NewEngine()
	}

	return nil
}

// addMetaFlags adds common arguments defined on meta to the command cmd.
// All commands that include meta have to call this!
func (m *meta) addMetaFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.IntVar(&m.timeout, "timeout", 10, "timeout for http client when retrieving sources")
	f.StringVarP(&m.file, "file", "f", ".", "file or directory to run configuration for")
}

// getExtraArgs returns extra terraform arguments, but filters out invalid arguments
func getExtraArgs(invalidArgs ...string) []string {
	extraArgs := []string{}
	for _, arg := range terraformArgs {
		invalidArg := false
		arg = fmt.Sprintf("-%s", arg)

		for _, ia := range invalidArgs {
			if strings.HasPrefix(arg, ia) {
				invalidArg = true
			}
		}

		if !invalidArg {
			extraArgs = append(extraArgs, arg)
		}
	}

	return extraArgs
}
