package semantic

import "testing"

func TestFieldRegistryDoesNotExposeInternalOnlyFields(t *testing.T) {
	forbidden := map[string]bool{
		"actionParams":     true,
		"conditionParams":  true,
		"groupComponent":   true,
		"groupComponentId": true,
		"repeatType":       true,
		"repeatValue":      true,
	}
	for _, mapping := range FieldRegistry() {
		if forbidden[mapping.Public] {
			t.Fatalf("internal-only field %q leaked as public mapping in domain %q", mapping.Public, mapping.Domain)
		}
	}
}

func TestFieldRegistryIncludesIntentExplainPublicFields(t *testing.T) {
	required := []string{
		FieldIntent,
		FieldImplemented,
		FieldPayloadGuide,
		FieldPayloadShape,
		FieldAcceptedFields,
		FieldNextStep,
	}
	seen := map[string]bool{}
	for _, mapping := range FieldRegistry() {
		seen[mapping.Public] = true
	}
	for _, field := range required {
		if !seen[field] {
			t.Fatalf("public field %q missing from field registry", field)
		}
	}
}

func TestProductIdentityUsesCapabilityPID(t *testing.T) {
	if FieldCapabilityProductID != "capabilityPid" {
		t.Fatalf("FieldCapabilityProductID = %q", FieldCapabilityProductID)
	}
	if InternalField(DomainProduct, FieldCapabilityProductID) != internalProductID {
		t.Fatalf("capability PID should map to internal pid")
	}
}

func TestPropertyRegistryMapsCommonProAbbreviations(t *testing.T) {
	tests := map[string]string{
		"m":    FieldMode,
		"o":    FieldOnline,
		"mv":   "motionDetected",
		"oc":   "occupancyDetected",
		"dc":   "doorClosed",
		"act":  "sensorActive",
		"alm":  "alarm",
		"sp":   FieldSwitchPower,
		"actt": "airConditionerTargetTemperature",
		"pt":   "productType",
		"psk":  "wiFiPassword",
		"dt":   "curtainManualOperationAllowed",
		"pi":   "curtainStartPositionCalibrated",
		"pe":   "curtainEndPositionCalibrated",
		"hk":   "homeKitLinked",
		"mfl":  "matterLinked",
		"dver": "daliVersion",
		"dpt":  "daliSwitchType",
		"pf":   "powerFactor",
	}
	for input, want := range tests {
		if got := PropertyName(input); got != want {
			t.Fatalf("PropertyName(%q) = %q, want %q", input, got, want)
		}
	}
	for _, sensitive := range []string{"psk", "wiFiPassword", "ltk", "localToken", "deviceKey"} {
		if !PropertySensitive(sensitive) {
			t.Fatalf("PropertySensitive(%q) = false", sensitive)
		}
	}
	if _, ok := LightPropertyID("mv"); ok {
		t.Fatalf("sensor motion must not become a direct light write property")
	}
}

func TestPropertyCatalogDoesNotExposeKnownInternalIDsAsPublicNames(t *testing.T) {
	forbidden := map[string]bool{
		"p": true, "l": true, "ct": true, "c": true, "m": true, "o": true,
		"mv": true, "oc": true, "dc": true, "act": true, "alm": true,
		"dt": true, "pi": true, "pe": true, "hk": true, "mfl": true,
		"dver": true, "dpt": true, "pf": true, "ddt": true, "gtin": true,
		"mock": true, "mimac": true, "ctRdy": true, "runSpeedRdy": true,
	}
	for _, mapping := range PropertyCatalog() {
		if forbidden[mapping.PublicName] {
			t.Fatalf("internal property id %q leaked as public name for id %q", mapping.PublicName, mapping.ID)
		}
	}
}

func TestNormalizeActionIgnoresCallerInternalFields(t *testing.T) {
	row := NormalizeAction(map[string]any{
		"targetType": "device",
		"targetId":   "50018330",
		"set": map[string]any{
			"power":            true,
			"brightness":       60,
			"colorTemperature": 3000,
			"p":                false,
			"l":                1,
			"ct":               6500,
		},
		"typeId": 99,
		"resId":  "bad-device",
		"params": map[string]any{
			"set": map[string]any{"p": false},
		},
	}, ActionOptions{})
	if row[internalTypeID] != ResourceDevice {
		t.Fatalf("typeId = %#v", row[internalTypeID])
	}
	if row[internalResourceID] != 50018330 {
		t.Fatalf("resId = %#v", row[internalResourceID])
	}
	params, ok := row[internalParams].(map[string]any)
	if !ok {
		t.Fatalf("params = %#v", row[internalParams])
	}
	set, ok := params[FieldSet].(map[string]any)
	if !ok {
		t.Fatalf("set = %#v", params[FieldSet])
	}
	if set[internalPower] != true || set[internalBrightness] != 60 || set[internalColorTemperature] != 3000 {
		t.Fatalf("set = %#v", set)
	}
	if set[internalPower] == false || set[internalBrightness] == 1 || set[internalColorTemperature] == 6500 {
		t.Fatalf("caller internal short key overrode public set: %#v", set)
	}
	public := ToPublicAction(row)
	publicSet, ok := public[FieldSet].(map[string]any)
	if !ok {
		t.Fatalf("public set = %#v", public[FieldSet])
	}
	for _, internal := range []string{internalPower, internalBrightness, internalColorTemperature} {
		if _, ok := publicSet[internal]; ok {
			t.Fatalf("internal property leaked in public set: %#v", publicSet)
		}
	}
}

func TestToPublicLightSetMapsPropertyVocabularyAndDropsRawShortKeys(t *testing.T) {
	public := ToPublicLightSet(map[string]any{
		"p":           true,
		"l":           60,
		"ct":          3000,
		"c":           16711680,
		"mv":          true,
		"oc":          true,
		"1-p":         false,
		"2-mv":        true,
		"psk":         "secret",
		"unknown_raw": "drop",
		"effectName":  "sunrise",
	})
	if public[FieldPower] != true || public[FieldBrightness] != 60 || public[FieldColorTemperature] != 3000 || public[FieldColor] != 16711680 {
		t.Fatalf("public light set did not map common fields: %#v", public)
	}
	if public["motionDetected"] != true || public["occupancyDetected"] != true || public["1-power"] != false || public["2-motionDetected"] != true {
		t.Fatalf("public light set did not map extended property vocabulary: %#v", public)
	}
	for _, forbidden := range []string{"p", "l", "ct", "c", "mv", "oc", "1-p", "2-mv", "psk", "unknown_raw"} {
		if _, ok := public[forbidden]; ok {
			t.Fatalf("raw or sensitive property %q leaked in public set: %#v", forbidden, public)
		}
	}
	if public["effectName"] != "sunrise" {
		t.Fatalf("descriptive custom key should be preserved: %#v", public)
	}
}

func TestNormalizeActionRejectsInternalOnlyActionShape(t *testing.T) {
	row := NormalizeAction(map[string]any{
		"typeId": 2,
		"resId":  "50018330",
		"params": map[string]any{
			"set": map[string]any{"p": true},
		},
	}, ActionOptions{})
	if len(row) != 0 {
		t.Fatalf("internal-only action should not produce public normalized action: %#v", row)
	}
}

func TestToPublicConditionMapsInternalPropertyToStandardName(t *testing.T) {
	condition := ToPublicCondition(map[string]any{
		"typeId":  2,
		"resId":   "50018330",
		"resName": "人在传感器",
		"prop":    "mv",
		"op":      "eq",
		"value":   true,
		"source":  "sensor",
	})
	if condition[FieldProperty] != "motionDetected" {
		t.Fatalf("property = %#v, condition = %#v", condition[FieldProperty], condition)
	}
	if condition[FieldTargetType] != "device" || condition[FieldTargetID] != "50018330" || condition[FieldTargetName] != "人在传感器" {
		t.Fatalf("target fields = %#v", condition)
	}
	for _, internal := range []string{"typeId", "resId", "resName", "prop"} {
		if _, ok := condition[internal]; ok {
			t.Fatalf("internal condition field %q leaked: %#v", internal, condition)
		}
	}
}

func TestNormalizeAutomationConditionMapsPublicEventFields(t *testing.T) {
	condition := NormalizeAutomationCondition(map[string]any{
		FieldConditionKind:       "event",
		FieldTargetType:          "device",
		FieldTargetKey:           "slot-1",
		FieldCapabilityProductID: 198666,
		FieldEventID:             42,
		FieldEventArgs:           map[string]any{"arg1": 423},
		FieldProperty:            FieldBrightness,
		FieldOperation:           "gt",
		FieldValue:               10,
	})
	if condition[internalConditionKind] != "event" || condition[internalTempID] != "slot-1" || condition[internalTypeID] != ResourceDevice || condition[internalProductID] != 198666 || condition[FieldID] != 42 || condition[internalProperty] != internalBrightness {
		t.Fatalf("condition = %#v", condition)
	}
	if args := condition[internalEventArgs].(map[string]any); args["arg1"] != 423 {
		t.Fatalf("event args = %#v", args)
	}
}

func TestToPublicConditionMapsAutomationInternalsToClearFields(t *testing.T) {
	condition := ToPublicCondition(map[string]any{
		"type":    "event",
		"typeId":  2,
		"tempId":  "slot-1",
		"pid":     198666,
		"id":      42,
		"extArgs": map[string]any{"arg1": 423},
		"prop":    "mv",
	})
	if condition[FieldConditionKind] != "event" || condition[FieldTargetType] != "device" || condition[FieldTargetKey] != "slot-1" || condition[FieldCapabilityProductID] != 198666 || condition[FieldEventID] != 42 || condition[FieldProperty] != "motionDetected" {
		t.Fatalf("condition = %#v", condition)
	}
	if args := condition[FieldEventArgs].(map[string]any); args["arg1"] != 423 {
		t.Fatalf("event args = %#v", args)
	}
	for _, internal := range []string{"type", "typeId", "tempId", "pid", "id", "extArgs", "prop"} {
		if _, ok := condition[internal]; ok {
			t.Fatalf("internal condition field %q leaked: %#v", internal, condition)
		}
	}
}

func TestNormalizeProductIgnoresCallerInternalIdentityFields(t *testing.T) {
	product := NormalizeProduct(map[string]any{
		FieldSKUCode:             "1-000002044",
		FieldCapabilityProductID: 198666,
		FieldProductComponentID:  4,
		"materialCode":           "bad-code",
		"pid":                    1,
		"pcId":                   2,
	})
	if product[internalProductCode] != "1-000002044" || product[internalProductID] != 198666 || product[internalProductCategoryID] != 4 {
		t.Fatalf("product = %#v", product)
	}
}
