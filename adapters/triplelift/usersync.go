package triplelift

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewTripleliftSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("triplelift", 28, temp, adapters.SyncTypeRedirect)
}
