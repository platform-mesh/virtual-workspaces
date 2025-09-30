package authentication

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/authentication/request/union"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"
)

func New(restCfg *rest.Config) authenticator.Request {
	cfg := rest.CopyConfig(restCfg)

	cfg.CertData = nil
	cfg.KeyData = nil

	client, err := rest.HTTPClientFor(restCfg)
	if err != nil {
		panic(err)
	}

	parsedURL, err := url.Parse(restCfg.Host)
	if err != nil {
		panic(err)
	}

	return union.NewFailOnError(
		WorkspaceAuthenticator,
		bearertoken.New(OIDCAuthenticator(client, fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host))),
	)
}

func OIDCAuthenticator(client *http.Client, baseURL string) authenticator.Token {
	return authenticator.TokenFunc(func(ctx context.Context, token string) (*authenticator.Response, bool, error) {
		clusterPath, ok := ctx.Value(clusterPath{}).(string)
		if !ok {
			return &authenticator.Response{}, false, fmt.Errorf("no cluster path in context")
		}

		requestURL := fmt.Sprintf("%s/clusters/%s/version", baseURL, clusterPath)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			return nil, false, err
		}

		req.Header.Set("Authorization", "Bearer "+token)

		res, err := client.Do(req)
		if err != nil {
			return nil, false, err
		}
		defer res.Body.Close()

		switch res.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusForbidden:
			// one could also continue here and use the OIDC userinfo endpoint to get more information about the user
			// but for now, just having a valid token is enough to be considered authenticated
			// even if the user does not have permissions to do anything (403)
			// this is similar to how the kube-apiserver handles authentication
			// we map all valid tokens to the "system:authenticated" group
			return &authenticator.Response{
				User: &user.DefaultInfo{
					Name:   "system:anonymous",
					Groups: []string{"system:authenticated"},
				},
			}, true, nil
		default:
			return &authenticator.Response{}, false, fmt.Errorf("unexpected status code %d from %s", res.StatusCode, requestURL)
		}
	})
}

type clusterPath struct{}

var WorkspaceAuthenticator = authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
	segments := strings.Split(req.URL.Path, "/")

	idx := slices.Index(segments, "clusters")
	if idx == -1 || idx+1 >= len(segments) {
		return &authenticator.Response{}, false, fmt.Errorf("no cluster path in request URL %q", req.URL.Path)
	}

	*req = *req.WithContext(context.WithValue(req.Context(), clusterPath{}, segments[idx+1]))

	return &authenticator.Response{}, false, nil
})
