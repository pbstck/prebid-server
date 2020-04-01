package pubstack

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"

	"github.com/prebid/prebid-server/analytics"
)

type payload struct {
	request  openrtb.BidRequest
	response openrtb.BidResponse
}

const (
	MAX_BUFF_EVENT_COUNT = 4
	MAX_BUFF_SIZE_BYTES  = 2 * 1000000
	BUFF_TIMEOUT_MINUTES = 15

	AUCTION = "auction"
)

//Module that can perform transactional logging
type PubstackModule struct {
	c      chan []byte
	intake string
	scope  string
}

func sendBuffer(wr *gzip.Writer, buff *bytes.Buffer, intake, targetPath string) (*bytes.Buffer, *gzip.Writer, error) {
	wr.Close()
	payload := make([]byte, buff.Len())
	_, err := buff.Read(payload)
	if err != nil {
		return nil, nil, err
	}
	buff = bytes.NewBufferString("")
	wr = gzip.NewWriter(buff)

	target, _ := url.Parse(intake)
	target.Path = path.Join(target.Path, targetPath)
	return buff, wr, sendPayloadToTarget(payload, target.String())
}

func handleBuffer(c chan []byte, intake string) {
	buff := bytes.NewBufferString("")
	gzWriter := gzip.NewWriter(buff)

	tick := time.NewTicker(BUFF_TIMEOUT_MINUTES * time.Minute)
	eventCount := 0

	var err error

	for {
		select {
		case evt := <-c:
			// add \n to event
			evt = append(evt, byte('\n'))
			gzWriter.Write(evt)
			gzWriter.Flush()
			eventCount++

			// we received an event and the buffer is full
			if eventCount == MAX_BUFF_EVENT_COUNT || buff.Len() == MAX_BUFF_SIZE_BYTES {
				buff, gzWriter, err = sendBuffer(gzWriter, buff, intake, AUCTION)
				if err != nil {
					glog.Error(err)
				}
				eventCount = 0
			}

		case <-tick.C:
			// buffer has not been cleaned for the too long
			if eventCount > 0 {
				buff, gzWriter, err = sendBuffer(gzWriter, buff, intake, AUCTION)
				if err != nil {
					glog.Error(err)
				}
				eventCount = 0
			}
		}
	}
}

//Writes AuctionObject to file
func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	// send openrtb request
	payload, err := jsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize auction")
		return
	}
	p.c <- payload
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
	glog.Infof("Initializing pubstack module with scope: %s intake %s\n", scope, intake)

	URL, err := url.Parse(intake)
	if err != nil {
		glog.Errorf("Fail to initialize pubstack analytics: %s", err.Error())
		return nil, fmt.Errorf("endpoint url is invalid")
	}

	if err := testEndpoint(URL); err != nil {
		glog.Errorf("Fail to initialize pubstack analytics: %s", err.Error())
		return nil, fmt.Errorf("fail to reach endpoint")
	}

	chn := make(chan []byte)

	go handleBuffer(chn, intake)

	// path is overriden by testEndpoint
	return &PubstackModule{
		chn,
		intake,
		scope,
	}, nil
}
