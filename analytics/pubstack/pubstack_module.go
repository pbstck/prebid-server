package pubstack

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/analytics/clients"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/docker/go-units"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"github.com/prebid/prebid-server/analytics/pubstack/helpers"

	"github.com/prebid/prebid-server/analytics"
)

type Configuration struct {
	ScopeId  string          `json:"scopeId"`
	Endpoint string          `json:"endpoint"`
	Features map[string]bool `json:"features"`
}

// routes for events
const (
	auction    = "auction"
	cookieSync = "cookiesync"
	amp        = "amp"
	setUiud    = "setuid"
	video      = "video"
)

type bufferConfig struct {
	timeout time.Duration
	count   int64
	size    int64
}

type PubstackModule struct {
	eventChannels map[string]*eventchannel.EventChannel
	configCh      chan *Configuration
	scope         string
	cfg           *Configuration
	buffsCfg      *bufferConfig
	mux           sync.Mutex
}

func NewPubstackModule(scope, endpoint, configRefreshDelay string, maxEventCount int, maxByteSize, maxTime string) (analytics.PBSAnalyticsModule, error) {
	glog.Infof("[pubstack] Initializing module scope=%s endpoint=%s\n", scope, endpoint)

	// parse args

	refreshDelay, err := time.ParseDuration(configRefreshDelay)
	if err != nil {
		glog.Errorf("[pubstack] Fail to parse the module args, arg=analytics.pubstack.configuration_refresh_delay; %v", err)
		return nil, err
	}

	bufferCfg, err := newBufferConfig(maxEventCount, maxByteSize, maxTime)
	if err != nil {
		glog.Errorf("[pubstack] Fail to parse the module args, arg=analytics.pubstack.buffers; %v", err)
		return nil, err
	}

	defaultFeatures := make(map[string]bool)
	defaultFeatures[auction] = true
	defaultFeatures[video] = false
	defaultFeatures[amp] = false
	defaultFeatures[cookieSync] = false
	defaultFeatures[setUiud] = false

	defaultConfig := &Configuration{
		ScopeId:  scope,
		Endpoint: endpoint,
		Features: defaultFeatures,
	}

	pb := PubstackModule{
		scope:    scope,
		cfg:      defaultConfig,
		buffsCfg: bufferCfg,
		configCh: make(chan *Configuration),
	}

	pb.initEventChannels()
	go pb.start()

	// update periodically the config
	go pb.fetchAndUpdateConfig(refreshDelay)

	glog.Info("[pubstack] Pubstack analytics configured and ready")
	return &pb, nil
}

func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	if !p.isFeatureEnable(auction) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize auction")
		return
	}

	p.logObject(auction, payload)
}

func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	if !p.isFeatureEnable(video) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.logObject(video, payload)
}

func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	if !p.isFeatureEnable(setUiud) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.logObject(setUiud, payload)
}

func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	if !p.isFeatureEnable(cookieSync) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.logObject(cookieSync, payload)
}

func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	if !p.isFeatureEnable(amp) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAmpObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.logObject(amp, payload)
}

func (p *PubstackModule) fetchAndUpdateConfig(waitPeriod time.Duration) {
	// TODO it's working?
	end := make(chan os.Signal)
	signal.Notify(end, os.Interrupt, syscall.SIGTERM)

	tick := time.NewTicker(waitPeriod)

	for {
		select {
		case <-tick.C:
			config, err := fetchConfig(p.cfg.ScopeId, p.cfg.Endpoint)
			if err != nil {
				glog.Errorf("[pubstack] fail to fetch configuration: %v", err)
				continue
			}
			p.configCh <- config
		case <-end:
			return
		}
	}
}

func (p *PubstackModule) start() {
	end := make(chan os.Signal)
	signal.Notify(end, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case config := <-p.configCh:
			fmt.Printf("%v\n", config)
			p.mux.Lock()
			p.cfg = config
			p.initEventChannels()
			p.mux.Unlock()
		case <-end:
			return
		}
	}

}

func fetchConfig(scope string, intake string) (*Configuration, error) {
	u, err := url.Parse(intake)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "bootstrap")
	q, _ := url.ParseQuery(u.RawQuery)

	q.Add("scopeId", scope)
	u.RawQuery = q.Encode()

	res, err := clients.GetDefaultHttpInstance().Get(u.String())
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func newBufferConfig(count int, size, duration string) (*bufferConfig, error) {
	pDuration, err := time.ParseDuration(duration)
	if err != nil {
		return nil, err
	}
	pSize, err := units.FromHumanSize(size)
	if err != nil {
		return nil, err
	}
	return &bufferConfig{
		pDuration,
		int64(count),
		pSize,
	}, nil
}

func buildEndpointSender(baseUrl string, module string) eventchannel.Sender {
	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		glog.Fatal(err)
	}
	endpoint.Path = path.Join(endpoint.Path, "intake", module)
	return eventchannel.NewHttpSender(endpoint.String())
}

func (p *PubstackModule) initEventChannels() {
	if p.eventChannels == nil {
		p.eventChannels = make(map[string]*eventchannel.EventChannel)
	}
	p.eventChannels[amp] = eventchannel.NewEventChannel(buildEndpointSender(p.cfg.Endpoint, amp), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	p.eventChannels[auction] = eventchannel.NewEventChannel(buildEndpointSender(p.cfg.Endpoint, auction), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	p.eventChannels[cookieSync] = eventchannel.NewEventChannel(buildEndpointSender(p.cfg.Endpoint, cookieSync), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	p.eventChannels[video] = eventchannel.NewEventChannel(buildEndpointSender(p.cfg.Endpoint, video), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	p.eventChannels[setUiud] = eventchannel.NewEventChannel(buildEndpointSender(p.cfg.Endpoint, setUiud), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
}

func (p *PubstackModule) isFeatureEnable(feature string) bool {
	val, ok := p.cfg.Features[feature]
	if !ok || !val {
		return false
	}
	return true
}

func (p *PubstackModule) logObject(feature string, payload []byte) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.eventChannels[feature].Push(payload)
}
