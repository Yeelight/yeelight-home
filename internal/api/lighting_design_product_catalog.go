package api

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed assets/lighting_design_products.json
var lightingDesignProductCatalogJSON []byte

// lightingDesignProductEntry is a compact, release-safe subset extracted from
// Yeelight smart product records for design-slot identity and fallback matching.
type lightingDesignProductEntry struct {
	MaterialCode         string                          `json:"materialCode"`
	PID                  int64                           `json:"pid"`
	PCID                 int64                           `json:"pcId"`
	ConnectType          int                             `json:"connectType"`
	ProductName          string                          `json:"productName"`
	ProductSKU           string                          `json:"productSku"`
	ProductSPU           string                          `json:"productSpu"`
	Category             string                          `json:"category"`
	Series               string                          `json:"series"`
	Priority             int                             `json:"priority"`
	AdjustableProperties []string                        `json:"adjustableProperties"`
	PropertyEvents       []string                        `json:"propertyEvents"`
	SensorEvents         []string                        `json:"sensorEvents"`
	DesignAttributes     lightingDesignProductAttributes `json:"designAttributes"`
	DesignRoles          []string                        `json:"designRoles"`
	DesignKeywords       []string                        `json:"designKeywords"`
	CapabilityTags       []string                        `json:"capabilityTags"`
	Aliases              []string                        `json:"aliases"`
}

type lightingDesignProductAttributes struct {
	Colors        []string `json:"colors"`
	InstallStyles []string `json:"installStyles"`
	BeamAngles    []string `json:"beamAngles"`
	Openings      []string `json:"openings"`
	Sizes         []string `json:"sizes"`
	HeadCounts    []string `json:"headCounts"`
	Wattages      []string `json:"wattages"`
	Shapes        []string `json:"shapes"`
}

var lightingDesignProductCatalog = mustLoadLightingDesignProductCatalog()

func mustLoadLightingDesignProductCatalog() []lightingDesignProductEntry {
	var entries []lightingDesignProductEntry
	if err := json.Unmarshal(lightingDesignProductCatalogJSON, &entries); err != nil {
		panic(fmt.Sprintf("load lighting design product catalog: %v", err))
	}
	return entries
}
