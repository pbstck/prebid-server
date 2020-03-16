package pubstack

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"net/url"
	"path"

	"github.com/prebid/prebid-server/analytics"
)

type payload struct {
	request  openrtb.BidRequest
	response openrtb.BidResponse
}

const (
	AUCTION = "auction"
)

//Module that can perform transactional logging
type PubstackModule struct {
	intake *url.URL
	scope  string
}

//Writes AuctionObject to file
func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	// send openrtb request
	payload, err := jsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize auction")
		return
	}

	p.intake.Path = path.Join(p.intake.Path, AUCTION)
	err = sendPayloadToTarget(payload, p.intake.String())
	if err != nil {
		glog.Warning("Issues while sending auction object to the intake")
	}
}

//Writes VideoObject to file
func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	return
}

//Logs SetUIDObject to file
func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	return
}

//Logs CookieSyncObject to file
func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	return
}

//Logs AmpObject to file
func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	return
}

//Method to initialize the analytic module
func NewPubstackModule(scope, intake string) (analytics.PBSAnalyticsModule, error) {
	glog.Info("Initializing listener")
	glog.Infof("scope: %s intake %s\n", scope, intake)

	URL, err := url.Parse(intake)
	if err != nil {
		glog.Errorf("Fail to initialize pubstack analytics: %s", err.Error())
		return nil, fmt.Errorf("endpoint url is invalid")
	}

	if err := testEndpoint(URL); err != nil {
		glog.Errorf("Fail to initialize pubstack analytics: %s", err.Error())
		return nil, fmt.Errorf("fail to reach endpoint")
	}
	// path is overriden by testEndpoint
	URL, _ = url.Parse(intake)
	return &PubstackModule{
		URL,
		scope,
	}, nil
}
