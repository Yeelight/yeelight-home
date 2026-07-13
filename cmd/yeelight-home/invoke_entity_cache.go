package main

import (
	"context"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

type entityLoadOptions struct {
	PreferCache bool
	Refresh     bool
}

type entityResolveResult struct {
	Entities   api.EntityListResult
	Match      api.EntitySummary
	Candidates []api.EntitySummary
	MatchedBy  string
}

func (app *app) loadEntities(ctx context.Context, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, options entityLoadOptions) (api.EntityListResult, error) {
	if options.PreferCache && !options.Refresh {
		if cached, ok := app.topologyCache.Load(profile, region, houseID, time.Now()); ok {
			return cached, nil
		}
	}
	result, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return result, err
	}
	if houseID != "" {
		if saveErr := app.topologyCache.Save(profile, region, houseID, result, time.Now()); saveErr != nil {
			result.Warnings = appendWarning(result.Warnings, "topology_cache_save_failed")
		}
	}
	return result, nil
}

func (app *app) resolveEntity(ctx context.Context, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, target entityGetTarget) (entityResolveResult, error) {
	entities, err := app.loadEntities(ctx, endpoint, profile, region, houseID, authorization, clientID, entityLoadOptions{PreferCache: true})
	if err != nil {
		return entityResolveResult{}, err
	}
	match, candidates, matchedBy := findEntity(target, entities.Entities)
	if match.ID != "" || len(candidates) > 0 || topologyCacheHits(entities) == 0 {
		return entityResolveResult{Entities: entities, Match: match, Candidates: candidates, MatchedBy: matchedBy}, nil
	}
	refreshed, err := app.loadEntities(ctx, endpoint, profile, region, houseID, authorization, clientID, entityLoadOptions{Refresh: true})
	if err != nil {
		return entityResolveResult{Entities: entities, Match: match, Candidates: candidates, MatchedBy: matchedBy}, nil
	}
	match, candidates, matchedBy = findEntity(target, refreshed.Entities)
	return entityResolveResult{Entities: refreshed, Match: match, Candidates: candidates, MatchedBy: matchedBy}, nil
}

func (app *app) refreshTopologyCache(ctx context.Context, endpoint api.Endpoint, recordProfile string, recordRegion string, houseID string, authorization string, clientID string) (api.EntityListResult, error) {
	return app.loadEntities(ctx, endpoint, recordProfile, recordRegion, houseID, authorization, clientID, entityLoadOptions{Refresh: true})
}

func (app *app) refreshTopologyCacheAfterWrite(ctx context.Context, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, response contract.Response) contract.Response {
	if response.Status != "success" && response.Status != "partial" {
		return response
	}
	houseID := topologyRefreshHouseID(record, response)
	if strings.TrimSpace(houseID) == "" || operation.IsAccountScope(houseID) {
		return response
	}
	if record.Intent == "home.delete" {
		if err := app.topologyCache.Invalidate(record.Profile, record.Region, houseID); err != nil {
			response.Warnings = appendWarning(response.Warnings, "topology_cache_invalidate_failed")
		}
		return response
	}
	if entities, ok := response.Internal[semantic.FieldVerifiedTopology].(api.EntityListResult); ok && entities.Total > 0 {
		if saveErr := app.topologyCache.Save(record.Profile, record.Region, houseID, entities, time.Now()); saveErr != nil {
			response.Warnings = appendWarning(response.Warnings, "topology_cache_save_failed")
			return response
		}
		if response.Metrics == nil {
			response.Metrics = map[string]any{}
		}
		response.Metrics[semantic.FieldTopologyCacheRefreshCalls] = 0
		response.Metrics[semantic.FieldTopologyCacheWriteSource] = "write_verification"
		return response
	}
	if !intentAffectsTopologyCache(record.Intent) {
		return response
	}
	entities, err := app.refreshTopologyCache(ctx, endpoint, record.Profile, record.Region, houseID, authorization, clientID)
	if err != nil {
		response.Warnings = appendWarning(response.Warnings, "topology_cache_refresh_failed")
		return response
	}
	if response.Metrics == nil {
		response.Metrics = map[string]any{}
	}
	response.Metrics[semantic.FieldTopologyCacheRefreshCalls] = entityListAPICalls(entities)
	response.Metrics[semantic.FieldTopologyCacheWriteSource] = "post_write_refresh"
	return response
}

func intentAffectsTopologyCache(intent string) bool {
	switch intent {
	case "operation.batch.configure",
		"room.create",
		"room.rename",
		"room.update",
		"room.batch_create",
		"room.batch_update",
		"room.area.configure",
		"room.delete",
		"room.batch_delete",
		"area.create",
		"area.update",
		"area.delete",
		"area.batch_delete",
		"device.rename",
		"device.move",
		"device.move_room.batch",
		"device.remove",
		"device.unbind",
		"entity.rename.batch",
		"group.create",
		"group.update",
		"group.members.update",
		"group.delete",
		"group.batch_delete",
		"scene.create",
		"scene.update",
		"scene.delete",
		"scene.batch_delete",
		"automation.create",
		"automation.update",
		"automation.enable",
		"automation.disable",
		"automation.delete",
		"automation.batch_delete",
		"lighting.design.import",
		"device.slot.create",
		"lighting.design.apply",
		"gateway.delete":
		return true
	default:
		return false
	}
}

func topologyRefreshHouseID(record operation.Prepared, response contract.Response) string {
	for _, value := range []any{
		response.Result[semantic.FieldHouseID],
		record.Payload[semantic.FieldHouseID],
		record.HouseID,
	} {
		if houseID := valueIDString(value); houseID != "" {
			return houseID
		}
	}
	return ""
}

func topologyCacheHits(result api.EntityListResult) int {
	for _, warning := range result.Warnings {
		if warning == "topology_cache_hit" {
			return 1
		}
	}
	return 0
}
