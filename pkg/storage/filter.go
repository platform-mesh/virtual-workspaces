package storage

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/kcp-dev/client-go/dynamic"
	"github.com/kcp-dev/kcp/pkg/virtual/framework/forwardingregistry"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v3"
	extensionapiv1alpha1 "github.com/openmfp/extension-manager-operator/api/v1alpha1"
	"github.com/platform-mesh/virtual-workspaces/api/v1alpha1"
	"github.com/platform-mesh/virtual-workspaces/pkg/config"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type clusterPathKey struct{}

func WithClusterPath(ctx context.Context, path logicalcluster.Path) context.Context {
	return context.WithValue(ctx, clusterPathKey{}, path)
}

func ClusterPathFrom(ctx context.Context) (logicalcluster.Path, bool) {
	path, ok := ctx.Value(clusterPathKey{}).(logicalcluster.Path)
	if !ok {
		return logicalcluster.Path{}, false
	}
	return path, true
}

func ContentConfigurationLookup(client dynamic.ClusterInterface, cfg config.ServiceConfig) forwardingregistry.StorageWrapper {
	return forwardingregistry.StorageWrapperFunc(func(resource schema.GroupResource, storage *forwardingregistry.StoreFuncs) {
		delegateLister := storage.ListerFunc
		storage.ListerFunc = func(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {

			// This lists the current workspace's objects
			result, err := delegateLister.List(ctx, options)
			if err != nil {
				return nil, err
			}

			ul, _ := result.(*unstructured.UnstructuredList)

			path, ok := ClusterPathFrom(ctx)
			if !ok {
				return nil, kerrors.NewBadRequest("cluster path not found in context")
			}

			apiBindings, err := client.Cluster(path).Resource(schema.GroupVersionResource{
				Group:    "apis.kcp.io",
				Version:  "v1alpha1",
				Resource: "apibindings",
			}).List(ctx, v1.ListOptions{})
			if err != nil {
				return nil, err
			}

			parentPath, _ := path.Parent()

			entityType := "account"
			if strings.HasSuffix(parentPath.String(), "orgs") {
				entityType = "main"
			}

			err = apiBindings.EachListItem(func(o runtime.Object) error {
				binding := o.(*unstructured.Unstructured)

				apiExportName, ok, err := unstructured.NestedString(binding.Object, "spec", "reference", "export", "name")
				if err != nil || !ok {
					return err
				}

				apiExportWorkspacePath, ok, err := unstructured.NestedString(binding.Object, "status", "apiExportClusterName")
				if err != nil || !ok {
					return err
				}

				exportCtx := genericapirequest.WithCluster(ctx, genericapirequest.Cluster{
					Name: logicalcluster.Name(apiExportWorkspacePath),
				})

				exportOpts := options.DeepCopy()
				exportOpts.LabelSelector = labels.SelectorFromValidatedSet(map[string]string{
					cfg.ContentForLabel: apiExportName,
					cfg.EntityLabel:     entityType,
				})

				apiExportCCs, err := delegateLister.List(exportCtx, exportOpts)
				if kerrors.IsNotFound(err) {
					return nil
				}

				if err != nil {
					return err
				}

				ul.Items = append(ul.Items, apiExportCCs.(*unstructured.UnstructuredList).Items...)

				return nil
			})
			if err != nil {
				return nil, err
			}

			providerCtx := genericapirequest.WithCluster(ctx, genericapirequest.Cluster{
				Name: logicalcluster.Name(cfg.ProviderWorkspaceID),
			})

			providerOpts := options.DeepCopy()
			providerOpts.LabelSelector = labels.SelectorFromValidatedSet(map[string]string{
				cfg.EntityLabel: entityType,
			})

			providerCCs, err := delegateLister.List(providerCtx, providerOpts)
			if err != nil {
				return nil, err
			}

			ul.Items = append(ul.Items, providerCCs.(*unstructured.UnstructuredList).Items...)

			return ul, nil
		}
	})
}

func setupVirtualWorkspaceClient(kubeconfigPath, serverURL, virtualWorkspacePath string) (*dynamic.ClusterClientset, error) {
	clientCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientCfg.QPS = -1 // Disable rate limiting for the client

	if serverURL != "" {
		clientCfg.Host = serverURL
	}

	parsed, _ := url.Parse(clientCfg.Host)
	pathSegments := strings.Split(virtualWorkspacePath, "/")
	parsed.Path = path.Join(pathSegments...)

	clientCfg.Host = parsed.String()

	return dynamic.NewForConfig(clientCfg)
}

func Marketplace(cfg config.ServiceConfig) forwardingregistry.StorageWrapper {
	providerMetadataClient, err := setupVirtualWorkspaceClient(cfg.Kubeconfig, cfg.ServerURL, cfg.ProviderMetadataVirtualWorkspacePath)
	if err != nil {
		// TODO: handle error properly
		return nil
	}

	apiExportClient, err := setupVirtualWorkspaceClient(cfg.Kubeconfig, cfg.ServerURL, cfg.APIExportVirtualWorkspacePath)
	if err != nil {
		// TODO: handle error properly
		return nil
	}

	return forwardingregistry.StorageWrapperFunc(func(resource schema.GroupResource, storage *forwardingregistry.StoreFuncs) {
		storage.ListerFunc = func(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {

			providers, err := providerMetadataClient.Cluster(logicalcluster.NewPath("*")).Resource(
				schema.GroupVersionResource{
					Group:    "core.openmfp.io",
					Version:  "v1alpha1",
					Resource: "providermetadatas",
				},
			).List(ctx, v1.ListOptions{})
			if err != nil {
				return nil, err
			}

			path, ok := ClusterPathFrom(ctx)
			if !ok {
				return nil, kerrors.NewBadRequest("cluster path not found in context")
			}

			parentPath, _ := path.Parent()

			entityType := "account"
			if strings.HasSuffix(parentPath.String(), "orgs") {
				entityType = "main"
			}

			var results v1alpha1.MarketplaceEntryList

			providers.EachListItem(func(o runtime.Object) error {

				var provider extensionapiv1alpha1.ProviderMetadata
				err := runtime.DefaultUnstructuredConverter.FromUnstructured(o.(*unstructured.Unstructured).Object, &provider)
				if err != nil {
					return err
				}

				rawExports, err := apiExportClient.Cluster(logicalcluster.NewPath("*")).Resource(
					schema.GroupVersionResource{
						Group:    "apis.kcp.io",
						Version:  "v1alpha1",
						Resource: "apiexports",
					},
				).List(ctx, v1.ListOptions{
					LabelSelector: labels.SelectorFromValidatedSet(map[string]string{
						cfg.ContentForLabel: provider.GetName(),
						cfg.EntityLabel:     entityType,
					}).String(),
				})
				if err != nil {
					return err
				}

				var exports apisv1alpha1.APIExportList
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(rawExports.Object, &exports)
				if err != nil {
					return err
				}

				for _, export := range exports.Items {
					results.Items = append(results.Items, v1alpha1.MarketplaceEntry{
						Spec: v1alpha1.MarketplaceEntrySpec{
							ProviderMetadata: *provider.Spec.DeepCopy(),
							APIExport:        *export.DeepCopy(),
							Installed:        false, // TODO: implement logic to determine if the entry is installed
						},
					})
				}

				return nil
			})

			return &results, nil
		}
	})
}
