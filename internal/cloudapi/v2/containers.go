package v2

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *apiHandlers) GetValidateContainer(ctx echo.Context, params GetValidateContainerParams) error {
	source := params.Source
	reachable, errMsg := checkContainerReachable(ctx.Request().Context(), source)

	resp := ContainerValidationResponse{
		Source:    source,
		Reachable: reachable,
	}
	if errMsg != "" {
		resp.Error = &errMsg
	}

	return ctx.JSON(http.StatusOK, resp)
}

type containerRef struct {
	registry   string
	repository string
	tag        string
}

func parseContainerRef(source string) (containerRef, error) {
	ref := containerRef{tag: "latest"}

	if at := strings.LastIndex(source, "@"); at != -1 {
		ref.tag = source[at+1:]
		source = source[:at]
	} else if colon := strings.LastIndex(source, ":"); colon != -1 {
		rest := source[colon+1:]
		if !strings.Contains(rest, "/") {
			ref.tag = rest
			source = source[:colon]
		}
	}

	parts := strings.SplitN(source, "/", 2)
	if len(parts) < 2 {
		return ref, fmt.Errorf("invalid container reference: missing registry or repository")
	}

	ref.registry = parts[0]
	ref.repository = parts[1]

	if ref.registry == "docker.io" {
		ref.registry = "registry-1.docker.io"
		if !strings.Contains(ref.repository, "/") {
			ref.repository = "library/" + ref.repository
		}
	}

	return ref, nil
}

func checkContainerReachable(ctx context.Context, source string) (bool, string) {
	ref, err := parseContainerRef(source)
	if err != nil {
		return false, err.Error()
	}

	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.registry, ref.repository, ref.tag)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create request: %s", err)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("failed to reach registry: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		token, err := fetchAnonymousToken(ctx, resp, ref)
		if err != nil {
			return false, fmt.Sprintf("container requires authentication: %s", err)
		}
		req2, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		req2.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json")
		req2.Header.Set("Authorization", "Bearer "+token)
		resp2, err := client.Do(req2)
		if err != nil {
			return false, fmt.Sprintf("failed to reach registry with token: %s", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			return true, ""
		}
		return false, fmt.Sprintf("container not found (HTTP %d)", resp2.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		return true, ""
	}

	return false, fmt.Sprintf("container not found (HTTP %d)", resp.StatusCode)
}

func fetchAnonymousToken(ctx context.Context, resp *http.Response, ref containerRef) (string, error) {
	challenge := resp.Header.Get("Www-Authenticate")
	if challenge == "" {
		return "", fmt.Errorf("no Www-Authenticate header in 401 response")
	}

	realm, service, scope := parseWwwAuthenticate(challenge, ref)
	if realm == "" {
		return "", fmt.Errorf("could not parse Www-Authenticate challenge")
	}

	tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}

	tokenResp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %s", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d", tokenResp.StatusCode)
	}

	var tokenData struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("failed to decode token response: %s", err)
	}

	token := tokenData.Token
	if token == "" {
		token = tokenData.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("empty token in response")
	}

	return token, nil
}

func parseWwwAuthenticate(header string, ref containerRef) (realm, service, scope string) {
	header = strings.TrimPrefix(header, "Bearer ")
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		val := strings.Trim(kv[1], "\"")
		switch kv[0] {
		case "realm":
			realm = val
		case "service":
			service = val
		case "scope":
			scope = val
		}
	}

	if scope == "" {
		scope = fmt.Sprintf("repository:%s:pull", ref.repository)
	}

	return realm, service, scope
}
