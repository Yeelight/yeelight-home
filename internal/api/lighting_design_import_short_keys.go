package api

import "strings"

func lightingDesignExpandShortKeys(value any) map[string]any {
	item, ok := mapFromAny(value)
	if !ok {
		return map[string]any{}
	}
	expanded := lightingDesignDeepRenameKeys(item, "")
	result, _ := mapFromAny(expanded)
	return result
}

func lightingDesignDeepRenameKeys(value any, path string) any {
	switch typed := value.(type) {
	case map[string]any:
		result := map[string]any{}
		for key, child := range typed {
			currentPath := key
			if path != "" {
				currentPath = path + "." + key
			}
			newKey := lightingDesignMappedKey(currentPath, key)
			result[newKey] = lightingDesignDeepRenameKeys(child, currentPath)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, child := range typed {
			result = append(result, lightingDesignDeepRenameKeys(child, path))
		}
		return result
	default:
		return typed
	}
}

func lightingDesignMappedKey(path string, key string) string {
	segments := strings.Split(path, ".")
	for index := range segments {
		if mapped := lightingDesignShortKeyMap[strings.Join(segments[index:], ".")]; mapped != "" {
			return mapped
		}
	}
	if mapped := lightingDesignShortKeyMap[key]; mapped != "" {
		return mapped
	}
	return key
}

var lightingDesignShortKeyMap = map[string]string{
	"tid":   "tempId",
	"n":     "name",
	"rl":    "roomList",
	"dl":    "deviceList",
	"gl":    "groupList",
	"al":    "areaList",
	"sl":    "sceneList",
	"atl":   "automationList",
	"rtids": "roomTempIdList",
	"dtids": "deviceTempIdList",
	"cid":   "componentId",
	"pid":   "pid",
	"mc":    "materialCode",
	"as":    "actions",
	"tpid":  "typeId",
	"rn":    "resName",
	"ap":    "params",
	"rk":    "rank",
	"st":    "startTime",
	"et":    "endTime",
	"rt":    "repeatType",
	"rv":    "repeatValue",
	"ps":    "params",
	"tp":    "type",
	"cs":    "conditions",
	"c":     "clock",
	"d":     "duration",
	"s":     "set",
	"ds":    "details",
	"i":     "index",
	"v":     "value",
	"s.c":   "c",
	"s.tp":  "tp",
	"ap.dl": "delay",
}
