package main

import (
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
)

const repeatTypeOnce = 1

func validateAutomationConditionGroups(params map[string]any, repeatType int) string {
	conditions, ok := params["conditions"].([]any)
	if !ok || len(conditions) == 0 {
		return "invalid_automation_params"
	}
	eventGroups := 0
	factGroups := 0
	for _, item := range conditions {
		group, ok := item.(map[string]any)
		if !ok {
			return "invalid_automation_params"
		}
		groupType := strings.TrimSpace(requestString(group["type"]))
		children, ok := group["conditions"].([]any)
		if ok {
			if groupType == "" || len(children) == 0 {
				return "invalid_automation_params"
			}
			classification, reason := classifyAutomationConditionGroup(children)
			if reason != "" {
				return reason
			}
			switch classification {
			case "event":
				eventGroups++
				if groupType == "and" || (repeatType == repeatTypeOnce && groupContainsOneOf(children, "event")) {
					return "invalid_automation_params"
				}
			case "fact":
				factGroups++
				if repeatType == repeatTypeOnce {
					return "invalid_automation_params"
				}
				if groupType != "and" && groupType != "or" {
					return "invalid_automation_params"
				}
			}
			if eventGroups > 1 || factGroups > 1 {
				return "invalid_automation_params"
			}
			continue
		}
		if strings.TrimSpace(requestString(group["type"])) == "" {
			return "invalid_automation_params"
		}
	}
	return ""
}

func validateAutomationConditionReferences(conditions []any, entities api.EntityListResult) string {
	for _, condition := range conditions {
		typed, ok := condition.(map[string]any)
		if !ok {
			return "invalid_automation_params"
		}
		if children, ok := typed["conditions"].([]any); ok {
			if reason := validateAutomationConditionReferences(children, entities); reason != "" {
				return reason
			}
		}
		if hasResourceReference(typed) {
			if reason := validateResourceReference(typed["typeId"], typed["resId"], entities, "invalid_automation_condition_resource_type", "invalid_automation_condition_reference"); reason != "" {
				return reason
			}
		}
	}
	return ""
}

func classifyAutomationConditionGroup(children []any) (string, string) {
	types := map[string]bool{}
	for _, child := range children {
		typed, ok := child.(map[string]any)
		if !ok {
			return "", "invalid_automation_params"
		}
		childType := strings.TrimSpace(requestString(typed["type"]))
		if childType == "" {
			return "", "invalid_automation_params"
		}
		types[childType] = true
	}
	if types["event"] || types["alarm"] || types["fact_change"] {
		return "event", ""
	}
	if len(types) != 1 || !types["fact"] {
		return "", "invalid_automation_params"
	}
	return "fact", ""
}

func groupContainsOneOf(children []any, expected ...string) bool {
	wanted := map[string]bool{}
	for _, value := range expected {
		wanted[value] = true
	}
	for _, child := range children {
		typed, ok := child.(map[string]any)
		if !ok {
			continue
		}
		if wanted[strings.TrimSpace(requestString(typed["type"]))] {
			return true
		}
	}
	return false
}
