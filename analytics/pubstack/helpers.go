package pubstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
)

type RequestType string

const (
	HEALTH = "/v1/health"
)

func testEndpoint(endpoint *url.URL) error {
	endpoint.Path = HEALTH

	r, err := http.Get(endpoint.String())
	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("receive code %d instead of %d", r.StatusCode, http.StatusOK)
	}
	return nil
}

func sendPayloadToTarget(payload []byte, endpoint string) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		glog.Error(err)
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
	}
	return nil
}

func jsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	type alias analytics.AuctionObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(ao),
	})

	if err == nil {
		return b, nil
	} else {
		return []byte(""), fmt.Errorf("transactional logs error: auction object badly formed %v", err)
	}
}
