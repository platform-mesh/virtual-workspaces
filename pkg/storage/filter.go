package storage

import (
	"context"
	"strings"

	"github.com/kcp-dev/client-go/dynamic"
	"github.com/kcp-dev/kcp/pkg/virtual/framework/forwardingregistry"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/platform-mesh/virtual-workspaces/pkg/config"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func Marketplace(client dynamic.ClusterInterface, cfg config.ServiceConfig) forwardingregistry.StorageWrapper {
	return forwardingregistry.StorageWrapperFunc(func(resource schema.GroupResource, storage *forwardingregistry.StoreFuncs) {
		delegateLister := storage.ListerFunc
		storage.ListerFunc = func(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {

			// This lists the current workspace's objects
			result, err := delegateLister.List(ctx, options)
			if err != nil {
				return nil, err
			}

			ul, _ := result.(*unstructured.UnstructuredList)

			path := ctx.Value("clusterPath").(logicalcluster.Path)

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
