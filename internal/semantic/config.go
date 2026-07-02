package semantic

const (
	ResourceRoom        = 1
	ResourceDevice      = 2
	ResourceCustomGroup = 3
	ResourceMeshGroup   = 4
	ResourceHome        = 5
	ResourceScene       = 6
	ResourceAutomation  = 12
)

type aliasConfig struct {
	TargetTypes        []string
	TargetIDs          []string
	TargetNames        []string
	TargetKeys         []string
	ActionParams       []string
	ActionSets         []string
	SubIndexes         []string
	DirectLightKeys    []string
	ConditionAliases   []string
	ConditionParams    []string
	TimeAliases        []string
	ConditionKinds     []string
	ProductCodes       []string
	ProductIDs         []string
	ProductCategoryIDs []string
	GroupComponents    []string
	SlotMembers        []string
}

var actionAliasConfig = aliasConfig{
	TargetTypes: []string{
		FieldTargetType,
	},
	TargetIDs: []string{
		FieldTargetID,
	},
	TargetNames: []string{
		FieldTargetName,
	},
	TargetKeys: []string{
		FieldTargetKey,
	},
	ActionParams: []string{},
	ActionSets: []string{
		FieldSet,
	},
	SubIndexes: []string{
		FieldSubIndex,
	},
	DirectLightKeys: []string{
		FieldPower, FieldBrightness, FieldColorTemperature, FieldColor,
	},
	ConditionAliases: []string{
		FieldConditionType,
	},
	ConditionParams: []string{
		FieldTrigger,
	},
	TimeAliases: []string{
		FieldTime,
	},
	ConditionKinds: []string{
		FieldConditionKind,
		FieldType,
	},
	ProductCodes: []string{
		FieldSKUCode,
		FieldProductCode,
	},
	ProductIDs: []string{
		FieldCapabilityProductID,
	},
	ProductCategoryIDs: []string{
		FieldProductComponentID,
		FieldProductCategoryID,
	},
	GroupComponents: []string{},
	SlotMembers: []string{
		FieldSlotKeys,
	},
}

var lightPropertyIDs = map[string]string{
	internalPower:            internalPower,
	FieldPower:               internalPower,
	"on":                     internalPower,
	"开关":                     internalPower,
	internalBrightness:       internalBrightness,
	FieldBrightness:          internalBrightness,
	"level":                  internalBrightness,
	"亮度":                     internalBrightness,
	internalColorTemperature: internalColorTemperature,
	FieldColorTemperature:    internalColorTemperature,
	"color_temperature":      internalColorTemperature,
	"colourTemperature":      internalColorTemperature,
	"colour_temperature":     internalColorTemperature,
	"色温":                     internalColorTemperature,
	internalColor:            internalColor,
	FieldColor:               internalColor,
	"colour":                 internalColor,
	"rgb":                    internalColor,
	"hex":                    internalColor,
	"颜色":                     internalColor,
}

var lightPropertyNames = map[string]string{
	internalPower:            FieldPower,
	internalBrightness:       FieldBrightness,
	internalColorTemperature: FieldColorTemperature,
	internalColor:            FieldColor,
}
