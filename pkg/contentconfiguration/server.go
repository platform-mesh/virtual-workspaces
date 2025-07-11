package contentconfiguration

import (
	"context"
	"path"
	"strings"

	"github.com/kcp-dev/client-go/dynamic"
	"github.com/kcp-dev/kcp/pkg/server/filters"
	"github.com/kcp-dev/kcp/pkg/virtual/framework"
	virtualworkspacesdynamic "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic"
	kcpapidefinition "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/apidefinition"
	virtualrootapiserver "github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/platform-mesh/virtual-workspaces/pkg/apidefinition"
	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	"github.com/platform-mesh/virtual-workspaces/pkg/proxy"
	"github.com/platform-mesh/virtual-workspaces/pkg/storage"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
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
			RootPathResolver: newPathResolver(clusterResolver, virtualWorkspaceBaseURL),
			Authorizer:       nil, //TODO: implement at a later point
			ReadyChecker:     framework.ReadyFunc(func() error { return nil }),
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
					storage.Marketplace(dynamicClient, cfg),
				)

				gvr := schema.GroupVersionResource{
					Group:    "core.openmfp.io",
					Version:  "v1alpha1",
					Resource: "contentconfigurations",
				}

				return apidefinition.NewSingleResourceProvider(mainConfig, gvr, &resourceSchema, storeageProvider), nil
			},
		},
	}
}

func newPathResolver(clusterResolver proxy.ClusterResolver, virtualWorkspaceBaseURL string) framework.RootPathResolverFunc {
	return func(urlPath string, requestContext context.Context) (accepted bool, prefixToStrip string, completedContext context.Context) {
		// Supported URLs look like this: /services/service/clusters/<cluster>/api…

		// Ignore anything that does not target this virtual workspace.
		if !strings.HasPrefix(urlPath, virtualWorkspaceBaseURL) {
			return false, "", requestContext
		}

		// trim the common prefix ("/services/service")
		relPath := strings.TrimPrefix(urlPath, virtualWorkspaceBaseURL)

		// URL should now be /clusters/<path>/api…

		// begin to find cluster name
		clustersPrefix := "/clusters/"
		if !strings.HasPrefix(relPath, clustersPrefix) {
			return false, "", requestContext
		}

		relPath = strings.TrimPrefix(relPath, clustersPrefix)

		// URL should now be <cluster>/api…

		// extract cluster name
		parts := strings.SplitN(relPath, "/", 2)
		path := logicalcluster.NewPath(parts[0])

		// determine cluster-local request URL
		realPath := "/"
		if len(parts) > 1 {
			realPath += parts[1]
		}

		// setup cluster in completed context
		var cluster genericapirequest.Cluster
		if path == logicalcluster.Wildcard {
			cluster = genericapirequest.Cluster{
				PartialMetadataRequest: filters.IsPartialMetadataRequest(requestContext),
				Wildcard:               true,
			}
		} else {
			var err error
			cluster, err = clusterResolver(requestContext, path)
			if err != nil {
				return false, "", requestContext
			}

		}

		completedContext = storage.WithClusterPath(requestContext, path)
		completedContext = genericapirequest.WithCluster(completedContext, cluster)

		// Inject a dummy object into the context which later is filled with real
		// data during the authorization process; this allows two function side-by-side
		// to share data in the same context.
		// completedContext = authorization.WithAttributeHolder(completedContext)

		return true, strings.TrimSuffix(urlPath, realPath), completedContext
	}
}
