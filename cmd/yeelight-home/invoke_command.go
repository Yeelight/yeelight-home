package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	localruntime "github.com/yeelight/yeelight-home/internal/runtime"
)

func (app *app) runInvoke(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "--stdin" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home invoke --stdin")
		return exitInvalidInput
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read stdin: %v\n", err)
		return exitInternalError
	}
	request, err := contract.DecodeRequest(data)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid SkillRequest: %v\n", err)
		return exitInvalidInput
	}
	response, err := app.invoke(context.Background(), request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invoke: %v\n", err)
		return exitInternalError
	}
	encoded, err := contract.EncodeResponse(response)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode SkillResponse: %v\n", err)
		return exitInternalError
	}
	if _, err := stdout.Write(encoded); err != nil {
		_, _ = fmt.Fprintf(stderr, "write stdout: %v\n", err)
		return exitInternalError
	}
	return exitOK
}

func (app *app) invoke(ctx context.Context, request contract.Request) (contract.Response, error) {
	flags := cliFlags{values: map[string]string{}}
	if region := requestString(request.Parameters["region"]); region != "" {
		flags.values["region"] = region
	}
	if houseID := requestHouseID(request); houseID != "" {
		flags.values["house-id"] = houseID
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return contract.Response{}, err
	}
	profile := context.Profile
	if !context.TokenPresent {
		return localruntime.NewEngine(false).Invoke(request), nil
	}
	if !isImplementedInvokeIntent(request.Intent) {
		return localruntime.NewEngine(true).Invoke(request), nil
	}
	region := context.Region
	clientID := context.ClientID
	houseID := context.HouseID
	endpoint := context.Endpoint
	accessToken := context.AccessToken
	switch request.Intent {
	case "account.info":
		return app.invokeAccountInfo(ctx, request, endpoint, accessToken, clientID)
	case "home.member.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, homeMemberListSpec())
	case "home.member.current.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.member.current.get", "已读取当前家庭成员的脱敏只读信息。"))
	case "device.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.detail.get", "已读取设备详情的安全摘要。"))
	case "device.attr.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.attr.list", "已读取设备属性的安全摘要。"))
	case "device.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.list", "已读取家庭设备候选列表。"))
	case "room.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.detail.get", "已读取房间详情的安全摘要。"))
	case "room.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.list", "已读取家庭房间列表。"))
	case "room.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.search", "已搜索家庭房间候选。"))
	case "area.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("area.detail.get", "已读取区域详情的安全摘要。"))
	case "home.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.detail.get", "已读取家庭详情的安全摘要。"))
	case "home.stat.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.stat.get", "已读取家庭统计的安全摘要。"))
	case "geo_area.children.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("geo_area.children.list", "已读取地理区域下级城市候选。"))
	case "geo_area.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("geo_area.search", "已按名称搜索地理区域候选。"))
	case "group.structure.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.structure.list", "已读取设备组结构的安全摘要。"))
	case "group.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.list", "已读取家庭分组列表。"))
	case "group.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.search", "已搜索家庭分组候选。"))
	case "group.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.detail.get", "已读取设备组详情的安全摘要。"))
	case "scene.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.detail.get", "已读取情景详情的安全摘要。"))
	case "scene.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.list", "已读取家庭情景列表。"))
	case "scene.scoped.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.scoped.list", "已按家庭或房间读取情景列表。"))
	case "scene.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.search", "已搜索家庭情景。"))
	case "automation.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.list", "已读取家庭自动化列表。"))
	case "automation.supported.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.supported.list", "已读取自动化支持能力。"))
	case "automation.supported.v2.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.supported.v2.list", "已读取自动化 V2 支持能力。"))
	case "automation.rule.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.rule.list", "已读取家庭规则列表。"))
	case "automation.list.page":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.list.page", "已分页读取自动化列表。"))
	case "automation.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.detail.get", "已读取自动化详情的安全摘要。"))
	case "schedule_job.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("schedule_job.list", "已读取家庭定时任务列表。"))
	case "message.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("message.list", "已读取当前账号消息列表的安全摘要。"))
	case "sensor.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("sensor.list", "已读取家庭传感器列表的安全摘要。"))
	case "sensor.event.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("sensor.event.list", "已读取传感器事件列表的安全摘要。"))
	case "device.energy.summary":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.energy.summary", "已读取设备用电摘要。"))
	case "device.weather.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.weather.get", "已读取设备天气上下文。"))
	case "device.virtual_count.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.virtual_count.get", "已读取家庭虚拟设备数量。"))
	case "meshgroup.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("meshgroup.detail.get", "已读取 Mesh 组详情的安全摘要。"))
	case "node.sorted_device.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("node.sorted_device.list", "已读取节点下设备排序列表。"))
	case "gateway.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.detail.get", "已读取网关详情的安全摘要。"))
	case "gateway.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.list", "已读取家庭网关列表的安全摘要。"))
	case "gateway.thread.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.thread.get", "已读取网关 Thread 信息的安全摘要。"))
	case "gateway.stats.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.stats.list", "已读取家庭网关统计的安全摘要。"))
	case "gateway.scene_relation.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.scene_relation.list", "已读取网关关联情景的安全摘要。"))
	case "home.summary":
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunListWithSelectedFallback(ctx, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		}, houseID)
		if err != nil {
			return contract.Response{}, err
		}
		return homeSummaryResponse(request, summary), nil
	case "home.list":
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunListWithSelectedFallback(ctx, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		}, houseID)
		if err != nil {
			return contract.Response{}, err
		}
		return homeListResponse(request, summary, "home-list-readonly", "已读取账号下家庭列表。"), nil
	case "home.search":
		if strings.TrimSpace(firstNonEmptyString(
			requestString(request.Parameters["fuzzyName"]),
			requestString(request.Parameters["name"]),
			requestString(request.Parameters["keyword"]),
			requestString(request.Parameters["query"]),
		)) == "" {
			return homeSearchClarificationResponse(request), nil
		}
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunSearch(ctx, request.Parameters, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		})
		if err != nil {
			return contract.Response{}, err
		}
		return homeListResponse(request, summary, "home-search-readonly", "已搜索账号下匹配家庭。"), nil
	case "entity.list":
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		return entityListResponse(request, entities), nil
	case "entity.get":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return entityGetClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		match, candidates, matchedBy := findEntity(target, entities.Entities)
		if match.ID == "" {
			return entityGetClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if len(candidates) > 1 && target.id == "" {
			return entityGetClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		return entityGetResponse(request, entities, match, matchedBy), nil
	case "entity.capabilities":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return entityCapabilitiesClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		match, candidates, _ := findEntity(target, entities.Entities)
		if match.ID == "" {
			return entityCapabilitiesClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if len(candidates) > 1 && target.id == "" {
			return entityCapabilitiesClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.Type == "device" {
			capabilities, err := api.NewDeviceCapabilitiesClient(endpoint, nil).Run(ctx, api.DeviceCapabilitiesRequest{
				HouseID:  houseID,
				DeviceID: match.ID,
				Credentials: api.DeviceCapabilitiesCredentials{
					Authorization: accessToken,
					ClientID:      clientID,
				},
			})
			if err == nil {
				return entityDeviceCapabilitiesResponse(request, entities, match, capabilities), nil
			}
			return entityCapabilitiesFallbackResponse(request, entities, match, "设备实例级 schema 读取失败，已降级为保守能力边界。"), nil
		}
		return entityCapabilitiesResponse(request, entities, match), nil
	case "state.query":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return stateQueryClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		match, candidates, _ := findEntity(target, entities.Entities)
		if match.ID == "" {
			return stateQueryClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if len(candidates) > 1 && target.id == "" {
			return stateQueryClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.Type != "device" {
			return stateQueryClarificationResponse(request, "target_not_device", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
		}
		propertyName := stateQueryPropertyName(request)
		propertySet := []string{}
		if propertyName == "" {
			if capabilities, err := api.NewDeviceCapabilitiesClient(endpoint, nil).Run(ctx, api.DeviceCapabilitiesRequest{
				HouseID:  houseID,
				DeviceID: match.ID,
				Credentials: api.DeviceCapabilitiesCredentials{
					Authorization: accessToken,
					ClientID:      clientID,
				},
			}); err == nil {
				propertySet = stateQueryPropertySet(capabilities.Device)
			}
		}
		state, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
			DeviceID:     match.ID,
			PropertyName: propertyName,
			PropertySet:  propertySet,
			Credentials: api.StateQueryCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		return stateQueryResponse(request, entities, match, state), nil
	case "scene.execute":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return sceneExecuteClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		match, candidates, _ := findEntity(target, entities.Entities)
		if match.ID == "" {
			return sceneExecuteClarificationResponse(request, "scene_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if len(candidates) > 1 && target.id == "" {
			return sceneExecuteClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.Type != "scene" {
			return sceneExecuteClarificationResponse(request, "target_not_scene", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
		}
		execution, err := api.NewSceneExecuteClient(endpoint, nil).Run(ctx, api.SceneExecuteRequest{
			HouseID: houseID,
			SceneID: match.ID,
			Credentials: api.SceneExecuteCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		return sceneExecuteResponse(request, entities, match, execution), nil
	case "scene.test":
		return app.invokeSceneTest(ctx, request, endpoint, houseID, accessToken, clientID)
	case "light.power.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, accessToken, clientID, lightPowerSpec())
	case "light.brightness.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, accessToken, clientID, lightBrightnessSpec())
	case "light.brightness.adjust":
		return app.invokeLightPropertyAdjust(ctx, request, endpoint, houseID, accessToken, clientID, lightBrightnessAdjustSpec())
	case "light.color_temperature.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, accessToken, clientID, lightColorTemperatureSpec())
	case "light.color_temperature.adjust":
		return app.invokeLightPropertyAdjust(ctx, request, endpoint, houseID, accessToken, clientID, lightColorTemperatureAdjustSpec())
	case "light.color.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, accessToken, clientID, lightColorSpec())
	case "behavior.execute":
		return app.invokeBehaviorExecute(ctx, request, endpoint, houseID, accessToken, clientID)
	case "lighting.experience.apply":
		return app.invokeLightingExperienceApply(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.device":
		return app.invokeDiagnoseDevice(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.gateway":
		return app.invokeDiagnoseGateway(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.scene":
		return app.invokeDiagnoseScene(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.automation":
		return app.invokeDiagnoseAutomation(ctx, request, endpoint, houseID, accessToken, clientID)
	case "automation.explain":
		return app.invokeAutomationExplain(ctx, request, endpoint, houseID, accessToken, clientID)
	case "automation.capabilities":
		return app.invokeMetadataLocalPlan(request, houseID, automationCapabilitiesSpec())
	case "panel.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, panelGetSpec())
	case "panel.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("panel.list", "已读取家庭下面板列表的安全摘要。"))
	case "panel.button.type.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("panel.button.type.get", "已按类型读取面板按键配置的安全摘要。"))
	case "screen.control.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("screen.control.list", "已读取屏控制设备列表的安全摘要。"))
	case "knob.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, knobGetSpec())
	case "upgrade.file.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.file.list", "已读取设备可用升级文件的安全摘要。"))
	case "upgrade.progress.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.progress.get", "已读取设备升级进度的安全摘要。"))
	case "upgrade.file.batch_list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.file.batch_list", "已批量读取设备可用升级文件的安全摘要。"))
	case "progress.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("progress.get", "已读取任务进度的安全摘要。"))
	case "app_upgrade.latest.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("app_upgrade.latest.get", "已读取 App 最新升级版本的安全摘要。"))
	case "ota.version_file.batch_list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("ota.version_file.batch_list", "已按版本批量读取 OTA 升级文件的安全摘要。"))
	case "node.property_config.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("node.property_config.get", "已读取节点属性配置的安全摘要。"))
	case "thing.schema.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.list", "已读取可见产品物模型列表的安全摘要。"))
	case "thing.schema.detail.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.detail.list", "已读取可见产品物模型详情列表的安全摘要。"))
	case "thing.schema.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.get", "已读取产品物模型详情的安全摘要。"))
	case "thing.schema.event.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.event.list", "已读取产品事件物模型的安全摘要。"))
	case "thing.product.info.batch_get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.info.batch_get", "已批量读取产品定义的安全摘要。"))
	case "thing.product.info.v3.batch_get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.info.v3.batch_get", "已按版本批量读取产品定义的安全摘要。"))
	case "thing.product.list.v3":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.list.v3", "已读取版本化产品列表的安全摘要。"))
	case "thing.product_domain.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_domain.list", "已读取产品域目录的安全摘要。"))
	case "thing.product_faq.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.list", "已读取产品帮助 FAQ 列表的安全摘要。"))
	case "thing.product_faq.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.detail.get", "已读取产品帮助 FAQ 详情的安全摘要。"))
	case "thing.product_faq.type.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.type.list", "已读取产品帮助 FAQ 类型列表。"))
	case "thing.product_faq.item_type.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.item_type.list", "已读取产品帮助 FAQ 项类型列表。"))
	case "thing.product_faq.locale.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.locale.list", "已读取产品帮助 FAQ 支持语言列表。"))
	case "thing.product_faq.page.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.page.list", "已分页读取产品帮助 FAQ 列表的安全摘要。"))
	case "thing.product_faq.page_detail.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.page_detail.list", "已分页读取产品帮助 FAQ 详情列表的安全摘要。"))
	case "thing.category.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.category.list", "已读取物模型品类列表的安全摘要。"))
	case "thing.component.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.component.list", "已读取物模型组件列表的安全摘要。"))
	case "thing.component.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.component.get", "已读取物模型组件详情的安全摘要。"))
	case "thing.property.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.property.list", "已读取物模型属性列表的安全摘要。"))
	case "device.storage.get":
		return app.invokeMetadataReadonlyProjection(ctx, request, endpoint, houseID, accessToken, clientID, deviceStorageGetSpec())
	case "ai_voice.list":
		return app.invokeMetadataReadonlyProjection(ctx, request, endpoint, houseID, accessToken, clientID, aiVoiceListSpec())
	case "ai_voice.product.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, metadataDetailReadonlySpec("ai_voice.product.list", "已读取支持 AI 语音能力的产品列表安全摘要。"))
	case "favorite.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, favoriteListSpec())
	case "favorite.plan":
		return app.invokeMetadataLocalPlan(request, houseID, favoritePlanSpec())
	case "home.sort.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, houseID, accessToken, clientID, homeSortListSpec())
	case "home.sort.configure", "favorite.add", "favorite.update", "favorite.delete", "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		return app.invokeHomeOrganizationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.member.invite", "home.member.accept_share", "home.member.configure", "home.member.remove", "home.member.transfer", "home.member.quit":
		return app.invokeHomeMemberPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.lock_all", "home.unlock_all":
		return app.invokeHomeLockPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.create":
		return app.invokeHomeCreatePlan(ctx, request, endpoint, profile, region, accessToken, clientID)
	case "home.update", "room.batch_create", "room.batch_update", "room.area.configure":
		return app.invokeHomeSpaceConfigurationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.rename", "room.update", "area.update", "device.rename", "device.move", "group.update":
		return app.invokeSpaceOrganizationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.move_room.batch":
		return app.invokeSpaceBatchOrganizationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.remove", "gateway.delete", "home.delete":
		return app.invokeDestructiveDeletePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.unbind":
		return app.invokeDeviceUnbindPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "gateway.configure":
		return app.invokeGatewayConfigurationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "entity.rename.batch":
		return app.invokeEntityBatchRenamePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.delete", "area.delete", "group.delete", "scene.delete", "automation.delete":
		return app.invokeMetadataDeletePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.batch_delete", "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		return app.invokeMetadataBatchDeletePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "panel.button.configure", "panel.button_event.update", "panel.button_event.batch_update", "panel.button_event.reset", "knob.configure", "knob.reset":
		return app.invokePanelConfigurationPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "lighting.design.plan":
		return app.invokeLightingDesignPlan(ctx, request, endpoint, houseID, accessToken, clientID)
	case "lighting.design.apply":
		return app.invokeLightingDesignApplyPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.create":
		return app.invokeRoomCreatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "area.create":
		return app.invokeMetadataCreatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, areaCreateSpec())
	case "group.create":
		return app.invokeMetadataCreatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, groupCreateSpec())
	case "scene.create":
		return app.invokeMetadataCreatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, sceneCreateSpec())
	case "scene.update":
		return app.invokeSceneUpdatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "automation.create":
		return app.invokeMetadataCreatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, automationCreateSpec())
	case "automation.update":
		return app.invokeAutomationUpdatePlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "automation.enable", "automation.disable":
		return app.invokeAutomationStatusPlan(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "memory.remember":
		return app.invokeMemoryRememberPlan(request, profile, region, houseID)
	case "memory.list":
		return app.invokeMemoryList(request, profile, houseID)
	case "memory.pause":
		return app.invokeMemoryPauseResume(request, profile, houseID, true)
	case "memory.resume":
		return app.invokeMemoryPauseResume(request, profile, houseID, false)
	case "memory.forget":
		return app.invokeMemoryForget(request, profile, houseID)
	case "recommendation.list":
		return app.invokeRecommendationList(request, profile, houseID)
	case "recommendation.feedback":
		return app.invokeRecommendationFeedback(request, profile, houseID)
	case "plan.commit":
		return app.invokePlanCommit(ctx, request, endpoint, profile, region, accessToken, clientID)
	case "plan.cancel":
		return app.invokePlanCancel(request, profile, region, houseID)
	case "execution.undo":
		return app.invokeExecutionUndo(request, profile, region, houseID)
	default:
		return localruntime.NewEngine(true).Invoke(request), nil
	}
}

func homeSummaryResponse(request contract.Request, summary api.HomeSummaryResult) contract.Response {
	houses := make([]any, 0, len(summary.Houses))
	for _, house := range summary.Houses {
		houses = append(houses, map[string]any{
			"id":   house.ID,
			"name": house.Name,
		})
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到 %d 个家庭。", summary.HouseCount),
		Result: map[string]any{
			"region":     summary.Region,
			"houseCount": summary.HouseCount,
			"houses":     houses,
			"source":     summary.Source,
		},
		Warnings: []string{},
		TraceID:  "home-summary-readonly",
		Metrics: map[string]any{
			"apiCalls":  firstPositive(summary.APICalls, 1),
			"cacheHits": 0,
		},
	}
}

func homeListResponse(request contract.Request, summary api.HomeSummaryResult, traceID string, message string) contract.Response {
	houses := make([]any, 0, len(summary.Houses))
	for _, house := range summary.Houses {
		item := map[string]any{
			"id":   house.ID,
			"name": house.Name,
		}
		for key, value := range map[string]string{
			"icon":     house.Icon,
			"desc":     house.Desc,
			"areaCode": house.AreaCode,
			"areaName": house.AreaName,
		} {
			if strings.TrimSpace(value) != "" {
				item[key] = value
			}
		}
		if len(house.Counts) > 0 {
			item["counts"] = house.Counts
		}
		houses = append(houses, item)
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("%s共 %d 个候选家庭。", message, summary.HouseCount),
		Result: map[string]any{
			"region":     summary.Region,
			"houseCount": summary.HouseCount,
			"houses":     houses,
			"source":     summary.Source,
		},
		Warnings: []string{},
		TraceID:  traceID,
		Metrics: map[string]any{
			"apiCalls":  firstPositive(summary.APICalls, 1),
			"cacheHits": 0,
		},
	}
}

func homeSearchClarificationResponse(request contract.Request) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请提供要搜索的家庭名称关键词。",
		Clarification: map[string]any{
			"reason":         "home_search_keyword_missing",
			"requiredFields": []string{"parameters.name"},
		},
		Warnings: []string{},
		TraceID:  "home-search-clarification",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
		},
	}
}

func firstPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
