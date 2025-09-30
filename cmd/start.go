package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kcp-dev/client-go/dynamic"
	kcpauthorization "github.com/kcp-dev/kcp/pkg/virtual/framework/authorization"
	"github.com/spf13/cobra"
	authentication "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/platform-mesh/virtual-workspaces/pkg/contentconfiguration"
	"github.com/platform-mesh/virtual-workspaces/pkg/marketplace"
	"github.com/platform-mesh/virtual-workspaces/pkg/path"
	"github.com/platform-mesh/virtual-workspaces/pkg/proxy"
	"github.com/platform-mesh/virtual-workspaces/pkg/storage"

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

		rootAPIServerConfig.Generic.Authentication.Authenticator = authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
			authHeader := req.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			pathresolver := path.NewPathResolver(proxy.NewClusterResolver(clusterClient), contentconfiguration.VirtualWorkspaceBaseURL())
			_, _, clusterCtx := pathresolver.ResolveRootPath(req.URL.Path, req.Context())
			clusterPath, _ := storage.ClusterPathFrom(clusterCtx)
			fmt.Println("clusterPath", clusterPath)

			// Create Token Review Request payload
			if token == "" {
				return nil, false, nil // No token provided
			}
			trr := &authentication.TokenReview{
				Spec: authentication.TokenReviewSpec{
					Token:     token,
					Audiences: []string{"default"},
				},
			}

			gvr := schema.GroupVersionResource{
				Group:    "authentication.k8s.io",
				Version:  "v1",
				Resource: "tokenreviews",
			}

			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(trr)
			if err != nil {
				return nil, false, err
			}

			result, err := dynamicClient.Resource(gvr).Cluster(clusterPath).Create(req.Context(), &unstructured.Unstructured{Object: unstructuredObj}, metav1.CreateOptions{})
			if err != nil {
				return nil, false, err
			}
			response, _ := json.Marshal(result.Object)
			fmt.Printf("%s\n", string(response))

			var createdTRR authentication.TokenReview
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, &createdTRR)
			if err != nil {
				return nil, false, err
			}

			if !createdTRR.Status.Authenticated {
				return nil, false, nil // Authentication failed
			}

			fmt.Println("=== Authentication Request ===")
			fmt.Printf("Method: %s\n", req.Method)
			fmt.Printf("URL: %s\n", req.URL.String())
			fmt.Printf("Remote Address: %s\n", req.RemoteAddr)

			// Print all HTTP headers
			fmt.Println("HTTP Headers:")
			for name, values := range req.Header {
				for _, value := range values {
					fmt.Printf("  %s: %s\n", name, value)
				}
			}

			return &authenticator.Response{
				Audiences: createdTRR.Status.Audiences,
				User: &user.DefaultInfo{
					Name:   createdTRR.Status.User.Username,
					UID:    createdTRR.Status.User.UID,
					Groups: createdTRR.Status.User.Groups,
				},
			}, true, nil
		})

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
