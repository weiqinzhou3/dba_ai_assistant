package asset

import "time"

type AssetType string

const (
	TypeDatabaseTarget AssetType = "DatabaseTarget"
)

type Selector struct {
	Project         string `json:"project"`
	Environment     string `json:"environment"`
	ServiceInstance string `json:"service_instance"`
}

type Asset struct {
	AssetID         string    `json:"asset_id"`
	AssetType       AssetType `json:"asset_type"`
	Project         string    `json:"project"`
	Environment     string    `json:"environment"`
	ServiceInstance string    `json:"service_instance"`
	CanonicalName   string    `json:"canonical_name"`
	Sensitivity     string    `json:"sensitivity,omitempty"`
	ConnectionRef   string    `json:"connection_ref,omitempty"`
}

type ResolvedAssetSet struct {
	AssetIDs       []string  `json:"asset_ids"`
	Assets         []Asset   `json:"assets"`
	MatchedExactly bool      `json:"matched_exactly"`
	AssetType      AssetType `json:"asset_type"`
	ResolvedAt     time.Time `json:"resolved_at"`
}
