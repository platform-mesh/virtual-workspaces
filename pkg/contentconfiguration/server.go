package contentconfiguration

import (
	"context"
	"path"

	"github.com/kcp-dev/client-go/dynamic"
	"github.com/kcp-dev/kcp/pkg/virtual/framework"
	virtualworkspacesdynamic "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic"
	kcpapidefinition "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/apidefinition"
	virtualrootapiserver "github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/platform-mesh/virtual-workspaces/pkg/apidefinition"
	"github.com/platform-mesh/virtual-workspaces/pkg/authorization"
	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	vwspath "github.com/platform-mesh/virtual-workspaces/pkg/path"
	"github.com/platform-mesh/virtual-workspaces/pkg/proxy"
	"github.com/platform-mesh/virtual-workspaces/pkg/storage"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	genericapiserver "k8s.io/apiserver/pkg/server"
)

var Name = "contentconfigurations"

func VirtualWorkspaceBaseURL() string {
	return path.Join("/services", "contentconfigurations")
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
				rawResourceSchema, err := dynamicClient.Cluster(logicalcluster.NewPath(cfg.ResourceSchemaWorkspace)).Resource(schema.GroupVersionResource{
					Group:    "apis.kcp.io",
					Version:  "v1alpha1",
					Resource: "apiresourceschemas",
				}).Get(context.TODO(), cfg.ResourceSchemaName, v1.GetOptions{})
				if err != nil {
					return nil, err
				}

				var resourceSchema apisv1alpha1.APIResourceSchema
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(rawResourceSchema.Object, &resourceSchema)
				if err != nil {
					return nil, err
				}

				storeageProvider := storage.CreateStorageProviderFunc(
					dynamicClient,
					storage.ContentConfigurationLookup(dynamicClient, cfg),
				)

				gvr := schema.GroupVersionResource{
					Group:    "ui.platform-mesh.io",
					Version:  "v1alpha1",
					Resource: "contentconfigurations",
				}

				return apidefinition.NewSingleResourceProvider(mainConfig, gvr, &resourceSchema, storeageProvider), nil
			},
		},
	}
}
