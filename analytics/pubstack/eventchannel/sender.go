package eventchannel

import (
	"bytes"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics/clients"
	"net/http"
)

type Sender = func(payload []byte)

func NewHttpSender(endpoint string) Sender {
	return func(payload []byte) {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			glog.Error(err)
			return
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := clients.GetDefaultHttpInstance().Do(req)
		if err != nil {
			return
		}
		if resp.StatusCode != http.StatusOK {
			glog.Errorf("[pubstack] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
			return
		}
	}
}
