package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/analytics"
)

// JsonifyAuctionObject helpers to serialize auction into json line
func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	type alias analytics.AuctionObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(ao),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return []byte(""), fmt.Errorf("transactional logs error: auction object badly formed %v", err)
}
