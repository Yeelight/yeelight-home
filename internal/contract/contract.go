package contract

import (
	"encoding/json"
	"fmt"
	"strings"
)

const Version = "1.0"

type Request struct {
	ContractVersion string           `json:"contractVersion"`
	RequestID       string           `json:"requestId"`
	SessionID       string           `json:"sessionId,omitempty"`
	Locale          string           `json:"locale"`
	Utterance       string           `json:"utterance"`
	Intent          string           `json:"intent"`
	HomeRef         map[string]any   `json:"homeRef,omitempty"`
	Targets         []map[string]any `json:"targets,omitempty"`
	Parameters      map[string]any   `json:"parameters,omitempty"`
	Context         map[string]any   `json:"conversationContext,omitempty"`
	Options         map[string]any   `json:"options,omitempty"`
}

type Response struct {
	ContractVersion string         `json:"contractVersion"`
	RequestID       string         `json:"requestId"`
	Status          string         `json:"status"`
	UserMessage     string         `json:"userMessage"`
	Result          map[string]any `json:"result,omitempty"`
	Clarification   map[string]any `json:"clarification,omitempty"`
	Execution       map[string]any `json:"execution,omitempty"`
	Memory          map[string]any `json:"memory,omitempty"`
	Recommendation  map[string]any `json:"recommendation,omitempty"`
	Warnings        []string       `json:"warnings"`
	TraceID         string         `json:"traceId,omitempty"`
	Metrics         map[string]any `json:"metrics,omitempty"`
	Error           *Error         `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func DecodeRequest(data []byte) (Request, error) {
	var request Request
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("decode request: %w", err)
	}
	if request.ContractVersion != Version {
		return Request{}, fmt.Errorf("unsupported contractVersion %q", request.ContractVersion)
	}
	if strings.TrimSpace(request.RequestID) == "" {
		return Request{}, fmt.Errorf("requestId is required")
	}
	if request.Locale != "zh-CN" {
		return Request{}, fmt.Errorf("locale must be zh-CN")
	}
	if strings.TrimSpace(request.Utterance) == "" {
		return Request{}, fmt.Errorf("utterance is required")
	}
	if !isKnownIntent(request.Intent) {
		return Request{}, fmt.Errorf("unsupported intent %q", request.Intent)
	}
	return request, nil
}

func EncodeResponse(response Response) ([]byte, error) {
	if response.ContractVersion == "" {
		response.ContractVersion = Version
	}
	if response.Warnings == nil {
		response.Warnings = []string{}
	}
	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode response: %w", err)
	}
	return append(data, '\n'), nil
}

func isKnownIntent(intent string) bool {
	switch intent {
	case "home.summary",
		"home.list",
		"home.search",
		"account.info",
		"home.member.list",
		"home.member.current.get",
		"device.detail.get",
		"device.attr.list",
		"device.list",
		"room.detail.get",
		"room.list",
		"room.search",
		"area.detail.get",
		"home.detail.get",
		"home.stat.get",
		"geo_area.children.list",
		"geo_area.search",
		"group.structure.list",
		"group.list",
		"group.search",
		"group.detail.get",
		"scene.detail.get",
		"scene.list",
		"scene.scoped.list",
		"scene.search",
		"automation.list",
		"automation.supported.list",
		"automation.supported.v2.list",
		"automation.rule.list",
		"automation.list.page",
		"automation.detail.get",
		"schedule_job.list",
		"message.list",
		"sensor.list",
		"sensor.event.list",
		"device.energy.summary",
		"device.weather.get",
		"device.virtual_count.get",
		"meshgroup.detail.get",
		"node.sorted_device.list",
		"gateway.detail.get",
		"gateway.list",
		"gateway.thread.get",
		"gateway.stats.list",
		"gateway.scene_relation.list",
		"entity.list",
		"entity.get",
		"entity.capabilities",
		"state.query",
		"light.power.set",
		"light.brightness.set",
		"light.brightness.adjust",
		"light.color_temperature.set",
		"light.color_temperature.adjust",
		"light.color.set",
		"lighting.experience.apply",
		"behavior.execute",
		"scene.execute",
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
		"group.delete",
		"group.batch_delete",
		"scene.create",
		"scene.update",
		"scene.delete",
		"scene.batch_delete",
		"scene.test",
		"automation.create",
		"automation.update",
		"automation.enable",
		"automation.disable",
		"automation.delete",
		"automation.batch_delete",
		"gateway.configure",
		"gateway.delete",
		"home.delete",
		"automation.explain",
		"automation.capabilities",
		"panel.get",
		"panel.list",
		"panel.button.type.get",
		"screen.control.list",
		"knob.get",
		"panel.button.configure",
		"panel.button_event.update",
		"panel.button_event.batch_update",
		"panel.button_event.reset",
		"knob.configure",
		"knob.reset",
		"upgrade.file.list",
		"upgrade.progress.get",
		"upgrade.file.batch_list",
		"progress.get",
		"app_upgrade.latest.get",
		"ota.version_file.batch_list",
		"node.property_config.get",
		"thing.schema.list",
		"thing.schema.detail.list",
		"thing.schema.get",
		"thing.schema.event.list",
		"thing.product.info.batch_get",
		"thing.product.info.v3.batch_get",
		"thing.product.list.v3",
		"product.pedia.search",
		"thing.product_domain.list",
		"thing.product_faq.list",
		"thing.product_faq.detail.get",
		"thing.product_faq.type.list",
		"thing.product_faq.item_type.list",
		"thing.product_faq.locale.list",
		"thing.product_faq.page.list",
		"thing.product_faq.page_detail.list",
		"thing.category.list",
		"thing.component.list",
		"thing.component.get",
		"thing.property.list",
		"device.storage.get",
		"ai_voice.list",
		"ai_voice.product.list",
		"favorite.list",
		"favorite.plan",
		"home.sort.list",
		"home.sort.configure",
		"favorite.add",
		"favorite.update",
		"favorite.delete",
		"favorite.batch_add",
		"favorite.batch_update",
		"favorite.batch_delete",
		"home.member.invite",
		"home.member.accept_share",
		"home.member.configure",
		"home.member.remove",
		"home.member.transfer",
		"home.member.quit",
		"home.lock_all",
		"home.unlock_all",
		"home.create",
		"home.update",
		"diagnose.device",
		"diagnose.gateway",
		"diagnose.scene",
		"diagnose.automation",
		"lighting.design.plan",
		"lighting.design.apply",
		"lighting.design.import",
		"device.slot.create",
		"memory.remember",
		"memory.list",
		"memory.forget",
		"memory.pause",
		"memory.resume",
		"recommendation.list",
		"recommendation.feedback",
		"operation.batch.configure":
		return true
	default:
		return false
	}
}
