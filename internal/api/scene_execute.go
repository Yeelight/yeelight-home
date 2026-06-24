package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SceneExecuteCredentials struct {
	Authorization string
	ClientID      string
}

type SceneExecuteRequest struct {
	HouseID     string
	SceneID     string
	Credentials SceneExecuteCredentials
}

type SceneExecuteResult struct {
	Region   string `json:"region"`
	HouseID  string `json:"houseId"`
	SceneID  string `json:"sceneId"`
	Source   string `json:"source"`
	RawShape string `json:"rawShape"`
	APICalls int    `json:"apiCalls"`
}

type SceneExecuteClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewSceneExecuteClient(endpoint Endpoint, client *http.Client) SceneExecuteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SceneExecuteClient{endpoint: endpoint, client: client}
}

func (client SceneExecuteClient) Run(ctx context.Context, request SceneExecuteRequest) (SceneExecuteResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	sceneID := strings.TrimSpace(request.SceneID)
	if houseID == "" {
		return SceneExecuteResult{}, fmt.Errorf("house id is required")
	}
	if sceneID == "" {
		return SceneExecuteResult{}, fmt.Errorf("scene id is required")
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/open/control/house/"+url.PathEscape(houseID)+"/control/w/scenes/"+url.PathEscape(sceneID), map[string]any{}, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return SceneExecuteResult{}, err
	}
	if !isBusinessOK(response) {
		return SceneExecuteResult{}, fmt.Errorf("scene execute returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return SceneExecuteResult{
		Region:   client.endpoint.Region,
		HouseID:  houseID,
		SceneID:  sceneID,
		Source:   "open_control_scene_endpoint",
		RawShape: responseDataType(response),
		APICalls: 1,
	}, nil
}
