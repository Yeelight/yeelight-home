package lanruntime

import (
	"strings"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

type toolRole string

const (
	roleList    toolRole = "list"
	roleDetail  toolRole = "detail"
	roleState   toolRole = "state"
	roleControl toolRole = "control"
	roleAction  toolRole = "action"
	roleFlow    toolRole = "flow"
	roleScene   toolRole = "scene"
)

type catalog struct {
	tools map[toolRole]lanmcp.Tool
}

func discoverCatalog(tools []lanmcp.Tool) catalog {
	result := catalog{tools: map[toolRole]lanmcp.Tool{}}
	for _, role := range []toolRole{roleList, roleDetail, roleState, roleControl, roleAction, roleFlow, roleScene} {
		bestScore := 0
		for _, tool := range tools {
			score := toolRoleScore(tool, role)
			if score > bestScore && schemaCanMapRole(tool.InputSchema, role) {
				bestScore = score
				result.tools[role] = tool
			}
		}
	}
	return result
}

func toolRoleScore(tool lanmcp.Tool, role toolRole) int {
	name := normalizedToolText(tool.Name)
	description := normalizedToolText(tool.Description)
	text := name + " " + description
	score := 0
	switch role {
	case roleList:
		if !strings.Contains(text, "list") || (!strings.Contains(text, "node") && !strings.Contains(text, "device")) {
			return 0
		}
		score += tokenScore(text, "list", "node", "device")
		if name == "list_nodes" {
			score += 20
		}
	case roleDetail:
		if name != "get_node" && name != "get_device" && !containsAny(text, "node detail", "device detail") {
			return 0
		}
		score += tokenScore(text, "get", "node", "device", "detail")
		if name == "get_node" {
			score += 20
		}
	case roleState:
		if !containsAny(text, "state", "properties", "property") || !containsAny(text, "get", "query", "read") {
			return 0
		}
		score += tokenScore(text, "get", "query", "read", "state", "properties", "node", "device")
		if strings.Contains(name, "list") {
			score -= 5
		}
	case roleControl:
		if !containsAny(text, "control", "set", "write", "execute") || !containsAny(text, "node", "device", "property", "action") {
			return 0
		}
		score += tokenScore(text, "control", "set", "write", "execute", "property", "node", "device", "action")
		if name == "control_node" {
			score += 20
		}
		if name == "execute_actions" {
			score += 20
		}
	case roleAction:
		if !containsAny(text, "execute", "run", "action") {
			return 0
		}
		score += tokenScore(text, "execute", "run", "action", "node", "device")
		if name == "execute_actions" {
			score += 20
		}
	case roleFlow:
		if containsAny(text, "flow", "effect") && containsAny(text, "execute", "run", "play") {
			score += tokenScore(text, "flow", "effect", "execute", "run", "play")
		} else if name == "execute_actions" && schemaHasArguments(tool.InputSchema, "actions", "operation", "capability", "value") {
			score += 20
		} else {
			return 0
		}
	case roleScene:
		if !strings.Contains(text, "scene") || !containsAny(text, "execute", "run") {
			return 0
		}
		score += tokenScore(text, "execute", "run", "scene")
		if name == "execute_scene" || name == "run_scene" {
			score += 20
		}
	}
	return score
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}

func tokenScore(text string, tokens ...string) int {
	score := 0
	for _, token := range tokens {
		if strings.Contains(text, token) {
			score += 3
		}
	}
	return score
}

func normalizedToolText(value string) string {
	return strings.ToLower(strings.NewReplacer("-", "_", ".", "_").Replace(strings.TrimSpace(value)))
}

func schemaCanMapRole(schema map[string]any, role toolRole) bool {
	if role == roleAction {
		return schemaHasAnyArgument(schema, "action", "actionname") || schemaHasArguments(schema, "actions", "operation", "capability")
	}
	if role == roleFlow {
		return schemaHasAnyArgument(schema, "flow") || schemaHasArguments(schema, "actions", "operation", "capability", "value")
	}
	properties := schemaProperties(schema)
	if len(properties) == 0 {
		return role == roleList && len(schemaRequired(schema)) == 0
	}
	for name, value := range properties {
		if mappableArgumentName(name, role) {
			return true
		}
		if len(schemaProperties(asMap(value))) > 0 && schemaCanMapRole(asMap(value), role) {
			return true
		}
	}
	return false
}

func mappableArgumentName(name string, role toolRole) bool {
	normalized := normalizedArgumentName(name)
	switch normalized {
	case "houseid", "homeid", "nodeid", "deviceid", "targetid", "entityid",
		"nodename", "devicename", "targetname", "entityname", "name", "roomname",
		"nodetype", "targettype", "entitytype", "type":
		return true
	case "id":
		return role == roleDetail
	case "property", "propertyname", "propname", "propertyid":
		return role == roleState || role == roleControl
	case "value", "properties", "propertyset", "params", "parameters", "actions", "action", "actionname", "command", "operation",
		"payload", "flow", "duration", "delay", "dryrun", "confirmsideeffect", "confirmed":
		return role == roleControl || role == roleAction || role == roleFlow
	case "requestid", "capability":
		return role == roleControl || role == roleAction || role == roleFlow
	case "sceneid", "scenename":
		return role == roleScene
	default:
		return false
	}
}

func schemaHasAnyArgument(schema map[string]any, names ...string) bool {
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[normalizedArgumentName(name)] = true
	}
	for name, value := range schemaProperties(schema) {
		propertySchema := asMap(value)
		if wanted[normalizedArgumentName(name)] || schemaHasAnyArgument(propertySchema, names...) || schemaHasAnyArgument(asMap(propertySchema["items"]), names...) {
			return true
		}
	}
	if itemSchema := asMap(schema["items"]); len(itemSchema) > 0 {
		return schemaHasAnyArgument(itemSchema, names...)
	}
	return false
}

func schemaHasArguments(schema map[string]any, names ...string) bool {
	for _, name := range names {
		if !schemaHasAnyArgument(schema, name) {
			return false
		}
	}
	return true
}
