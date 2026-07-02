package main

import (
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

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
	candidates := memoryPreferencesFromParameterList(request.Parameters[semantic.FieldPreferences])
	if len(candidates) == 0 {
		candidates = memoryPreferencesFromParameterList(request.Parameters[semantic.FieldMemories])
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
		scopeType:       firstNonEmptyString(firstRequestString(parameters, semantic.FieldScopeType), "home"),
		scopeRef:        firstRequestString(parameters, semantic.FieldScopeRef),
		preferenceType:  firstRequestString(parameters, semantic.FieldPreferenceType),
		preferenceValue: firstRequestString(parameters, semantic.FieldPreferenceValue),
		kind:            firstNonEmptyString(firstRequestString(parameters, semantic.FieldKind), "explicit"),
		status:          firstRequestString(parameters, semantic.FieldStatus),
		evidence:        firstRequestString(parameters, semantic.FieldEvidence),
	}
}
