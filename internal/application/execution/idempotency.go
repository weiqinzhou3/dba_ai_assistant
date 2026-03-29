package execution

import "fmt"

func BuildIdempotencyKey(actionName, targetAssetID, distinguishingParam string) string {
	return fmt.Sprintf("%s:%s:%s", actionName, targetAssetID, distinguishingParam)
}
