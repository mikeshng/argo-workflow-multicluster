package main

import (
	"context"
	"fmt"
	"os"

	goflag "flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"open-cluster-management.io/addon-framework/pkg/version"

	"open-cluster-management.io/argo-workflow-multicluster/addons/hub/install"
)

func main() {
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := newCommand()
	fmt.Printf("ArgoWorkflowInstallAddon version: %s\n", command.Version)

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "argoworkflow-install-addon",
		Short: "ArgoWorkflow Install Add-on",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			os.Exit(1)
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(newControllerCommand())

	return cmd
}

func newControllerCommand() *cobra.Command {
	cmd := controllercmd.
		NewControllerCommandConfig("argoworkflow-install-addon", version.Get(), runControllers).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the ArgoWorkflow install add-on controller"

	return cmd
}

func runControllers(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	return install.StartControllers(ctx, controllerContext.KubeConfig)
}
