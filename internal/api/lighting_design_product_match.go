package api

import (
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type lightingDesignProductMatch struct {
	Entry lightingDesignProductEntry
}

func lightingDesignExplicitProduct(slot map[string]any) lightingDesignProductMatch {
	entry := lightingDesignProductEntry{
		MaterialCode: firstAnyString(slot, semantic.ProductCodeFields()...),
		PID:          int64FromMap(slot, semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID), 0),
		PCID:         int64FromMap(slot, semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCategoryID), 0),
		ConnectType:  intFromMap(slot, semantic.FieldConnectType, -1),
		ProductName:  firstNonEmpty(stringFromMap(slot, semantic.FieldProductName), stringFromMap(slot, semantic.FieldName)),
		ProductSKU:   firstAnyString(slot, semantic.ProductSKUFields()...),
		ProductSPU:   firstAnyString(slot, semantic.ProductSPUFields()...),
		Category:     stringFromMap(slot, semantic.FieldCategory),
		Series:       stringFromMap(slot, semantic.FieldSeries),
	}
	if entry.MaterialCode != "" {
		for _, item := range lightingDesignProductCatalog {
			if strings.EqualFold(item.MaterialCode, entry.MaterialCode) {
				entry = item
				break
			}
		}
	}
	return lightingDesignProductMatch{Entry: entry}
}
