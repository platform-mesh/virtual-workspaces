package marketplace

import (
	"context"
	"os"
	"path"

	"github.com/kcp-dev/client-go/dynamic"
	"github.com/kcp-dev/kcp/pkg/virtual/framework"
	virtualworkspacesdynamic "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic"
	kcpapidefinition "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/apidefinition"
	virtualrootapiserver "github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	"github.com/platform-mesh/virtual-workspaces/pkg/apidefinition"
	"github.com/platform-mesh/virtual-workspaces/pkg/authorization"
	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	vwspath "github.com/platform-mesh/virtual-workspaces/pkg/path"
	"github.com/platform-mesh/virtual-workspaces/pkg/proxy"
	"github.com/platform-mesh/virtual-workspaces/pkg/storage"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	genericapiserver "k8s.io/apiserver/pkg/server"
)

var Name = "marketplace"

func VirtualWorkspaceBaseURL() string {
	return path.Join("/services", "marketplace")
}

func BuildVirtualWorkspace(
	cfg config.ServiceConfig,
	dynamicClient dynamic.ClusterInterface,
	kcpClusterClient kcpclientset.ClusterInterface,
	virtualWorkspaceBaseURL string,
) virtualrootapiserver.NamedVirtualWorkspace {

	clusterResolver := proxy.NewClusterResolver(kcpClusterClient)

	return virtualrootapiserver.NamedVirtualWorkspace{
		Name: Name,
		VirtualWorkspace: &virtualworkspacesdynamic.DynamicVirtualWorkspace{
			RootPathResolver: vwspath.NewPathResolver(clusterResolver, virtualWorkspaceBaseURL),
			Authorizer: authorization.NewAttributesKeeper(
				authorizer.AuthorizerFunc(func(ctx context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
					isAuthenticated := sets.New(a.GetUser().GetGroups()...).Has("system:authenticated")
					if isAuthenticated {
						return authorizer.DecisionAllow, "user is authenticated", nil
					}

					return authorizer.DecisionDeny, "user is not authenticated", nil
				}), // TODO: we can think of a bit more complex authorization logic, e.g. doing some SAR, for now it is better than nothing
			),
			ReadyChecker: framework.ReadyFunc(func() error { return nil }),
			BootstrapAPISetManagement: func(mainConfig genericapiserver.CompletedConfig) (kcpapidefinition.APIDefinitionSetGetter, error) {

				var resourceSchema apisv1alpha1.APIResourceSchema

				out, err := os.ReadFile("config/resources/apiresourceschema-marketplaceentries.marketplace.platform-mesh.io.yaml")
				if err != nil {
					return nil, err
				}

				err = yaml.Unmarshal(out, &resourceSchema)
				if err != nil {
					return nil, err
				}

				marketplaceFilter, err := storage.Marketplace(cfg)
				if err != nil {
					return nil, err
				}

				storeageProvider := storage.CreateStorageProviderFunc(
					dynamicClient,
					marketplaceFilter,
				)

				gvr := schema.GroupVersionResource{
					Group:    resourceSchema.Spec.Group,
					Version:  resourceSchema.Spec.Versions[0].Name,
					Resource: resourceSchema.Spec.Names.Plural,
				}

				return apidefinition.NewSingleResourceProvider(mainConfig, gvr, &resourceSchema, storeageProvider), nil
			},
		},
	}
}
