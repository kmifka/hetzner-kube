package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

func init() {
	declarePhaseCommand("install-masters", "install the control plane", func(cmd *cobra.Command, args []string) {
		clusterName := args[0]

		_, cluster := AppConf.Config.FindClusterByName(clusterName)
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)
		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)
		coordinator := pkg.NewProgressCoordinator()

		for _, node := range provider.GetAllNodes() {
			steps := 3
			if node.Name == masterNode.Name {
				steps += 4
				if len(provider.GetMasterNodes()) == 1 {
					steps += 1
				}
			} else {
				steps += 4
			}
			coordinator.StartProgress(node.Name, steps)
		}

		clusterManager := clustermanager.NewClusterManager(
			provider,
			AppConf.SSHClient,
			coordinator,
			clusterName,
			cluster.HaEnabled,
			cluster.IsolatedEtcd,
			cluster.CloudInitFile,
		)
		phase := phases.NewInstallMastersPhase(clusterManager)

		if phase.ShouldRun() {
			err := phase.Run()
			FatalOnError(err)
		}

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
	})
}
