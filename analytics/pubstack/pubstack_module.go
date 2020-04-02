package pubstack

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"github.com/prebid/prebid-server/analytics/pubstack/helpers"

	"github.com/prebid/prebid-server/analytics"
)

type payload struct {
	request  openrtb.BidRequest
	response openrtb.BidResponse
}

type Configuration struct {
	ScopeId        string `json:"scopeId"`
	CookieSync     bool   `json:"cookieSync"`
	Auction        bool   `json:"auction"`
	Amp            bool   `json:"AMP"`
	Video          bool   `json:"video"`
	SetUid         bool   `json:"setUid"`
	BufferSizeMega int64  `json:"buffSizeMega"`
	EventCount     int64  `json:"eventCount"`
	TimeoutMinutes int64  `json:"timeoutMinutes"`
}

const (
	MAX_BUFF_EVENT_COUNT = 4
	MAX_BUFF_SIZE_BYTES  = 2 * 1000000
	BUFF_TIMEOUT_MINUTES = 15

	AUCTION    = "auction"
	COOKIESYNC = "cookie_sync"
	AMP        = "amp"
	SETUID     = "setuid"
	VIDEO      = "video"
)

//Module that can perform transactional logging
type PubstackModule struct {
	chans map[string]*eventchannel.Channel
	scope string
	cfg   *Configuration
}

func getConfiguration(scope string, intake string) (*Configuration, error) {
	u, err := url.Parse(intake)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "bootstrap")
	q, _ := url.ParseQuery(u.RawQuery)

	q.Add("scopeId", scope)
	u.RawQuery = q.Encode()

	res, err := http.DefaultClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("fail to read payload body")
	}
	c := Configuration{}

	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	glog.Info(c)
	return &c, nil
}

//Writes AuctionObject to file
func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[AUCTION]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize auction")
		return
	}

	ch.Add(payload)
}

//Writes VideoObject to file
func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[VIDEO]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

//Logs SetUIDObject to file
func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[SETUID]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

//Logs CookieSyncObject to file
func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[VIDEO]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

//Logs AmpObject to file
func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[VIDEO]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAmpObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

//Method to initialize the analytic module
func NewPubstackModule(scope, intake string) (analytics.PBSAnalyticsModule, error) {
	glog.Infof("Initializing pubstack module with scope: %s intake %s\n", scope, intake)

	config, err := getConfiguration(scope, intake)
	if err != nil {
		glog.Errorf("Fail to initialize pubstack module due to %s\n", err.Error())
	}

	chanMap := make(map[string]*eventchannel.Channel)
	// enable auction forward

	if config.Amp {
		chanMap[AMP] = eventchannel.NewChannel(intake, AMP, config.BufferSizeMega, config.EventCount, time.Duration(config.TimeoutMinutes)*time.Minute)
	}
	if config.Auction {
		chanMap[AUCTION] = eventchannel.NewChannel(intake, AUCTION, config.BufferSizeMega, config.EventCount, time.Duration(config.TimeoutMinutes)*time.Minute)
	}
	if config.CookieSync {
		chanMap[COOKIESYNC] = eventchannel.NewChannel(intake, COOKIESYNC, config.BufferSizeMega, config.EventCount, time.Duration(config.TimeoutMinutes)*time.Minute)
	}
	if config.Video {
		chanMap[VIDEO] = eventchannel.NewChannel(intake, VIDEO, config.BufferSizeMega, config.EventCount, time.Duration(config.TimeoutMinutes)*time.Minute)
	}
	if config.SetUid {
		chanMap[SETUID] = eventchannel.NewChannel(intake, SETUID, config.BufferSizeMega, config.EventCount, time.Duration(config.TimeoutMinutes)*time.Minute)
	}

	return &PubstackModule{
		chanMap,
		scope,
		config,
	}, nil
}
