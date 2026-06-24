package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DevSeedRoomRequest struct {
	HouseID        string
	Name           string
	Description    string
	Icon           string
	AllowWriteDev  bool
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    DevSeedCredentials
}

type DevSeedRoomResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	RoomID     string `json:"roomId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
}

type roomSummary struct {
	ID     string
	Name   string
	Source string
}

func (client DevSeedClient) EnsureRoom(ctx context.Context, request DevSeedRoomRequest) (DevSeedRoomResult, error) {
	if client.endpoint.Region != "dev" {
		return DevSeedRoomResult{}, fmt.Errorf("dev seed is only allowed for dev region")
	}
	if !request.AllowWriteDev {
		return DevSeedRoomResult{}, fmt.Errorf("dev seed requires --allow-write-dev")
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return DevSeedRoomResult{}, fmt.Errorf("house id is required")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return DevSeedRoomResult{}, fmt.Errorf("room name is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DevSeedRoomResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	if err := client.verifyHouseScopedEntityList(ctx, houseID, credentials); err != nil {
		return DevSeedRoomResult{}, fmt.Errorf("verify house before room seed: %w", err)
	}
	existing, err := client.findRoomByName(ctx, houseID, name, credentials)
	if err != nil {
		return DevSeedRoomResult{}, err
	}
	if existing.ID != "" {
		return DevSeedRoomResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			RoomID:     existing.ID,
			Name:       existing.Name,
			Verified:   true,
			VerifiedBy: existing.Source,
		}, nil
	}
	created, err := client.createRoom(ctx, request, credentials)
	if err != nil {
		return DevSeedRoomResult{}, err
	}
	verified, err := client.verifyRoomByName(ctx, houseID, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	if err != nil {
		return DevSeedRoomResult{}, err
	}
	if verified.ID == "" {
		return DevSeedRoomResult{}, fmt.Errorf("unknown write result: created room was not found during verification; createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
	}
	return DevSeedRoomResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		RoomID:     verified.ID,
		Name:       verified.Name,
		Created:    true,
		Verified:   true,
		VerifiedBy: verified.Source,
	}, nil
}

func (client DevSeedClient) verifyRoomByName(ctx context.Context, houseID string, name string, credentials requestCredentials, attempts int, interval time.Duration) (roomSummary, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	for attempt := 0; attempt < attempts; attempt++ {
		room, err := client.findRoomByName(ctx, houseID, name, credentials)
		if err != nil || room.ID != "" || attempt == attempts-1 {
			return room, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return roomSummary{}, ctx.Err()
		case <-timer.C:
		}
	}
	return roomSummary{}, nil
}

func (client DevSeedClient) findRoomByName(ctx context.Context, houseID string, name string, credentials requestCredentials) (roomSummary, error) {
	room, _, err := RoomCreateClient{endpoint: client.endpoint, client: client.client}.findRoomByNameWithCallCount(ctx, houseID, name, credentials)
	if err != nil {
		return roomSummary{}, err
	}
	return room, nil
}

func (client DevSeedClient) createRoom(ctx context.Context, request DevSeedRoomRequest, credentials requestCredentials) (writeProbeResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	body, err := BuildRoomCreatePayload(houseID, request.Name, request.Description, request.Icon)
	if err != nil {
		return writeProbeResult{}, err
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+houseID+"/room/w/create", body, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("room create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}
