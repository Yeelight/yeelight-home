package main

func copyRequestParameters(parameters map[string]any) map[string]any {
	copied := make(map[string]any, len(parameters))
	for key, value := range parameters {
		copied[key] = value
	}
	return copied
}
