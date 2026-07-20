package i18n

import "testing"

func TestNormalizeRecognizesSupportedLocales(t *testing.T) {
	tests := map[string]string{
		"zh_CN.UTF-8": Chinese,
		"zh-Hans":     Chinese,
		"en":          English,
		"en-GB":       English,
	}
	for input, expected := range tests {
		actual, ok := Normalize(input)
		if !ok || actual != expected {
			t.Fatalf("Normalize(%q) = %q, %v; want %q, true", input, actual, ok, expected)
		}
	}
}

func TestTextUsesEnglishCatalog(t *testing.T) {
	message := Text(English, SceneExecuted, "Movie Time")
	if message != "Ran the scene: Movie Time." {
		t.Fatalf("message = %q", message)
	}
}

func TestTemplateFallsBackToEnglish(t *testing.T) {
	message := Text("unsupported", LightPowerSet, "Desk Lamp")
	if message != "Set the power state for Desk Lamp." {
		t.Fatalf("message = %q", message)
	}
}

func TestDetectUsesLocaleEnvironmentPriority(t *testing.T) {
	values := map[string]string{"LANG": "en_GB.UTF-8", "LC_MESSAGES": "zh_CN.UTF-8"}
	locale, ok := Detect(func(name string) (string, bool) {
		value, exists := values[name]
		return value, exists
	})
	if !ok || locale != Chinese {
		t.Fatalf("Detect = %q, %v", locale, ok)
	}
}
