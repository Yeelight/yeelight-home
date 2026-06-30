package main

import "github.com/yeelight/yeelight-home/internal/contract"

type memoryPreferenceCandidate struct {
	scopeType       string
	scopeRef        string
	preferenceType  string
	preferenceValue string
	kind            string
	status          string
	evidence        string
}

func memoryPreferenceFromRequest(request contract.Request) memoryPreferenceCandidate {
	candidates := memoryPreferencesFromRequest(request)
	if len(candidates) == 0 {
		return memoryPreferenceCandidate{}
	}
	return candidates[0]
}

func memoryPreferencesFromRequest(request contract.Request) []memoryPreferenceCandidate {
	candidates := memoryPreferencesFromParameterList(request.Parameters["preferences"])
	if len(candidates) == 0 {
		candidates = memoryPreferencesFromParameterList(request.Parameters["memories"])
	}
	if len(candidates) > 0 {
		return candidates
	}
	candidate := memoryPreferenceFromParameters(request.Parameters)
	if candidate.preferenceType != "" && candidate.preferenceValue != "" {
		return []memoryPreferenceCandidate{candidate}
	}
	return nil
}

func memoryPreferencesFromParameterList(value any) []memoryPreferenceCandidate {
	rawItems, ok := value.([]any)
	if !ok {
		return nil
	}
	candidates := make([]memoryPreferenceCandidate, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		candidate := memoryPreferenceFromParameters(item)
		if candidate.preferenceType == "" || candidate.preferenceValue == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func memoryPreferenceFromParameters(parameters map[string]any) memoryPreferenceCandidate {
	return memoryPreferenceCandidate{
		scopeType:       firstNonEmptyString(firstRequestString(parameters, "scopeType"), "home"),
		scopeRef:        firstRequestString(parameters, "scopeRef"),
		preferenceType:  firstRequestString(parameters, "preferenceType", "type"),
		preferenceValue: firstRequestString(parameters, "preferenceValue", "value"),
		kind:            firstNonEmptyString(firstRequestString(parameters, "kind"), "explicit"),
		status:          firstRequestString(parameters, "status"),
		evidence:        firstRequestString(parameters, "evidence"),
	}
}
