package cmd

import (
	"github.com/kcp-dev/client-go/dynamic"
	kcpauthorization "github.com/kcp-dev/kcp/pkg/virtual/framework/authorization"
	"github.com/spf13/cobra"

	"github.com/platform-mesh/virtual-workspaces/pkg/authentication"
	"github.com/platform-mesh/virtual-workspaces/pkg/contentconfiguration"
	"github.com/platform-mesh/virtual-workspaces/pkg/marketplace"

	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apiserver/pkg/authentication/request/union"
	genericapiserver "k8s.io/apiserver/pkg/server"

	virtualrootapiserver "github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	kcpclientset "github.com/kcp-dev/sdk/client/clientset/versioned/cluster"
)

var startCmd = &cobra.Command{
	Use: "start",
	RunE: func(cmd *cobra.Command, args []string) error {
		codecs := serializer.NewCodecFactory(scheme.Scheme)

		clientCfg, err := clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return err
		}

		if cfg.ServerURL != "" {
			clientCfg.Host = cfg.ServerURL
		}

		clientCfg.QPS = -1 // Disable rate limiting for the client

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

		err = delegatingAuthenticationOption.ApplyTo(&recommendedConfig.Authentication, recommendedConfig.SecureServing, recommendedConfig.OpenAPIConfig)
		if err != nil {
			return err
		}

		rootAPIServerConfig, err := virtualrootapiserver.NewConfig(recommendedConfig)
		if err != nil {
			return err
		}

		ctx := cmd.Context()

		rootAPIServerConfig.Extra.VirtualWorkspaces = []virtualrootapiserver.NamedVirtualWorkspace{
			contentconfiguration.BuildVirtualWorkspace(ctx, cfg, dynamicClient, clusterClient, contentconfiguration.VirtualWorkspaceBaseURL()),
			marketplace.BuildVirtualWorkspace(ctx, cfg, dynamicClient, clusterClient, marketplace.VirtualWorkspaceBaseURL()),
		}

		rootAPIServerConfig.Generic.Authentication.Authenticator = union.New(
			authentication.New(clientCfg),
			rootAPIServerConfig.Generic.Authentication.Authenticator,
		)

		rootAPIServerConfig.Generic.Authorization.Authorizer = kcpauthorization.NewVirtualWorkspaceAuthorizer(func() []virtualrootapiserver.NamedVirtualWorkspace {
			return rootAPIServerConfig.Extra.VirtualWorkspaces
		})

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
