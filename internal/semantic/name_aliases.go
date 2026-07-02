package semantic

import "strings"

type NameAlias struct {
	From string
	To   string
	Kind string
}

var nameAliases = []NameAlias{
	{From: "120度射灯", To: "E20射灯", Kind: "folk_name"},
	{From: "一二零射灯", To: "E20射灯", Kind: "folk_name"},
	{From: "艾斯系列", To: "S系列", Kind: "phonetic"},
	{From: "艾思系列", To: "S系列", Kind: "phonetic"},
	{From: "爱斯系列", To: "S系列", Kind: "phonetic"},
	{From: "爱思系列", To: "S系列", Kind: "phonetic"},
	{From: "人在感应器", To: "人在传感器", Kind: "synonym"},
	{From: "人体感应器", To: "人体传感器", Kind: "synonym"},
	{From: "门窗感应器", To: "门窗传感器", Kind: "synonym"},
	{From: "光照感应器", To: "光照传感器", Kind: "synonym"},
	{From: "网关", To: "gateway", Kind: "synonym"},
	{From: "120射灯", To: "E20射灯", Kind: "folk_name"},
	{From: "艾思", To: "S系列", Kind: "phonetic"},
	{From: "爱思", To: "S系列", Kind: "phonetic"},
	{From: "一来", To: "易来", Kind: "phonetic"},
	{From: "夜来", To: "易来", Kind: "phonetic"},
	{From: "干节点", To: "干接点", Kind: "synonym"},
	{From: "乾接点", To: "干接点", Kind: "synonym"},
	{From: "感应器", To: "传感器", Kind: "synonym"},
	{From: "开合帘", To: "窗帘", Kind: "folk_name"},
	{From: "小夜灯", To: "夜灯", Kind: "folk_name"},
	{From: "槽位", To: "设计槽位", Kind: "design_slot"},
	{From: "占位设备", To: "设计槽位", Kind: "design_slot"},
}

var normalizedNameAliases = normalizeNameAliases(nameAliases)

func NameAliases() []NameAlias {
	return append([]NameAlias(nil), nameAliases...)
}

func ApplyNameAliases(value string) string {
	normalized := normalizeNameTextBase(value)
	if normalized == "" {
		return ""
	}
	for _, alias := range normalizedNameAliases {
		if alias.From == "" || alias.To == "" {
			continue
		}
		normalized = strings.ReplaceAll(normalized, alias.From, alias.To)
	}
	return normalized
}

func normalizeNameAliases(aliases []NameAlias) []NameAlias {
	result := make([]NameAlias, 0, len(aliases))
	for _, alias := range aliases {
		result = append(result, NameAlias{
			From: normalizeNameTextBase(alias.From),
			To:   normalizeNameTextBase(alias.To),
			Kind: alias.Kind,
		})
	}
	return result
}
