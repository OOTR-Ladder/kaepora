package back

import (
	"kaepora/internal/util"
)

type Player struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
}
