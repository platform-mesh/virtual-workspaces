package cmd

import (
	"github.com/kcp-dev/client-go/dynamic"
	"github.com/platform-mesh/virtual-workspaces/pkg/contentconfiguration"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	genericapiserver "k8s.io/apiserver/pkg/server"

	virtualrootapiserver "github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
)

var startCmd = &cobra.Command{
	Use: "start",
	RunE: func(cmd *cobra.Command, args []string) error {
		codecs := serializer.NewCodecFactory(scheme.Scheme)

		clientCfg, err := clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return err
		}

		dynamicClient, err := dynamic.NewForConfig(clientCfg)
		if err != nil {
			return err
		}

		clusterClient, err := kcpclientset.NewForConfig(clientCfg)
		if err != nil {
			return err
		}

		recommendedConfig := genericapiserver.NewRecommendedConfig(codecs)

		err = secureServing.ApplyTo(&recommendedConfig.SecureServing)
		if err != nil {
			return err
		}

		rootAPIServerConfig, err := virtualrootapiserver.NewConfig(recommendedConfig)
		if err != nil {
			return err
		}

		rootAPIServerConfig.Extra.VirtualWorkspaces = []virtualrootapiserver.NamedVirtualWorkspace{
			contentconfiguration.BuildVirtualWorkspace(cfg, dynamicClient, clusterClient, contentconfiguration.VirtualWorkspaceBaseURL()),
		}

		completedRootAPIServerConfig := rootAPIServerConfig.Complete()
		rootAPIServer, err := virtualrootapiserver.NewServer(completedRootAPIServerConfig, genericapiserver.NewEmptyDelegate())
		if err != nil {
			return err
		}

		preparedRootAPIServer := rootAPIServer.GenericAPIServer.PrepareRun()
		if err := completedRootAPIServerConfig.WithOpenAPIAggregationController(preparedRootAPIServer.GenericAPIServer); err != nil {
			return err
		}

		return preparedRootAPIServer.RunWithContext(genericapiserver.SetupSignalContext())
	},
}
