package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	goflag "flag"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	utilflag "k8s.io/component-base/cli/flag"
	"open-cluster-management.io/addon-framework/pkg/version"
	hub "open-cluster-management.io/argo-workflow-multicluster/addons/hub/status_sync"
	spoke "open-cluster-management.io/argo-workflow-multicluster/addons/spoke/status_sync"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	var logger logr.Logger

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start up logger %v\n", err)
		os.Exit(1)
	}

	logger = zapr.NewLogger(zapLog)

	command := newCommand(logger)

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status-sync-addon",
		Short: "status-sync-addon",
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

	cmd.AddCommand(hub.NewManagerCommand("status-sync-addon", logger.WithName("status-manager")))
	cmd.AddCommand(spoke.NewAgentCommand("status-sync-addon", logger.WithName("status-agent")))

	return cmd
}
