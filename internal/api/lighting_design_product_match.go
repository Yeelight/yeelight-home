package api

import (
	"strings"
)

type lightingDesignProductMatch struct {
	Entry lightingDesignProductEntry
}

func lightingDesignExplicitProduct(slot map[string]any) lightingDesignProductMatch {
	entry := lightingDesignProductEntry{
		MaterialCode: firstNonEmpty(
			stringFromMap(slot, "materialCode"),
			stringFromMap(slot, "skuMaterialCode"),
		),
		PID:         int64FromMap(slot, "pid", 0),
		PCID:        int64FromMap(slot, "pcId", 0),
		ConnectType: intFromMap(slot, "connectType", -1),
		ProductName: firstNonEmpty(stringFromMap(slot, "productName"), stringFromMap(slot, "name")),
		ProductSKU:  firstNonEmpty(stringFromMap(slot, "productSku"), stringFromMap(slot, "sku")),
		ProductSPU:  firstNonEmpty(stringFromMap(slot, "productSpu"), stringFromMap(slot, "spu")),
		Category:    stringFromMap(slot, "category"),
		Series:      stringFromMap(slot, "series"),
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
