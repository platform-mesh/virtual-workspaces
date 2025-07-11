package apidefinition

import (
	"context"
	"fmt"

	kcpapidefinition "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/apidefinition"
	"github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/apiserver"
	dynamiccontext "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/context"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

type StorageProviderFunc func(ctx context.Context) (apiserver.RestProviderFunc, error)

type singleResourceAPIDefinitionSetProvider struct {
	config          genericapiserver.CompletedConfig
	gvr             schema.GroupVersionResource
	resource        *apisv1alpha1.APIResourceSchema
	storageProvider StorageProviderFunc
}

func NewSingleResourceProvider(
	config genericapiserver.CompletedConfig,
	gvr schema.GroupVersionResource,
	resource *apisv1alpha1.APIResourceSchema,
	storageProvider StorageProviderFunc,
) kcpapidefinition.APIDefinitionSetGetter {
	return &singleResourceAPIDefinitionSetProvider{
		config:          config,
		gvr:             gvr,
		resource:        resource,
		storageProvider: storageProvider,
	}
}

func (a *singleResourceAPIDefinitionSetProvider) GetAPIDefinitionSet(ctx context.Context, _ dynamiccontext.APIDomainKey) (apis kcpapidefinition.APIDefinitionSet, apisExist bool, err error) {
	restProvider, err := a.storageProvider(ctx)
	if err != nil {
		return nil, false, err
	}

	apiDefinition, err := apiserver.CreateServingInfoFor(
		a.config,
		a.resource,
		a.gvr.Version,
		restProvider,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create serving info: %w", err)
	}

	apis = kcpapidefinition.APIDefinitionSet{
		a.gvr: apiDefinition,
	}

	return apis, len(apis) > 0, nil
}

var _ kcpapidefinition.APIDefinitionSetGetter = &singleResourceAPIDefinitionSetProvider{}
