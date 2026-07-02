package semantic

import "strings"

type PropertyMapping struct {
	ID          string   `json:"id"`
	PublicName  string   `json:"publicName"`
	Description string   `json:"description,omitempty"`
	ValueType   string   `json:"valueType,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Sensitive   bool     `json:"sensitive,omitempty"`
}

var propertyIDAliases = buildPropertyIDAliases()
var propertyPublicNames = buildPropertyPublicNames()
var propertySensitiveIDs = buildPropertySensitiveIDs()

func PropertyID(value string) (string, bool) {
	id := propertyIDAliases[propertyKey(value)]
	return id, id != ""
}

func PropertyName(value string) string {
	if id, ok := PropertyID(value); ok {
		if name := propertyPublicNames[id]; name != "" {
			return name
		}
		return id
	}
	return strings.TrimSpace(value)
}

func PropertySensitive(value string) bool {
	id, ok := PropertyID(value)
	if !ok {
		id = strings.TrimSpace(value)
	}
	return propertySensitiveIDs[id]
}

func PropertyCatalog() []PropertyMapping {
	result := make([]PropertyMapping, len(propertyMappings))
	copy(result, propertyMappings)
	return result
}

func buildPropertyIDAliases() map[string]string {
	result := map[string]string{}
	for _, mapping := range propertyMappings {
		result[propertyKey(mapping.ID)] = mapping.ID
		result[propertyKey(mapping.PublicName)] = mapping.ID
		for _, alias := range mapping.Aliases {
			result[propertyKey(alias)] = mapping.ID
		}
	}
	return result
}

func buildPropertyPublicNames() map[string]string {
	result := map[string]string{}
	for _, mapping := range propertyMappings {
		result[mapping.ID] = mapping.PublicName
	}
	return result
}

func buildPropertySensitiveIDs() map[string]bool {
	result := map[string]bool{}
	for _, mapping := range propertyMappings {
		if mapping.Sensitive {
			result[mapping.ID] = true
		}
	}
	return result
}

func propertyKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", "\t", "", "\n", "")
	return replacer.Replace(value)
}
