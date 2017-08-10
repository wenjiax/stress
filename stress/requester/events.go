package requester

import (
	"net/http"
	"time"

	"github.com/wenjiax/stress/stress/reportor"
)

type Events struct {
	RequestBefore func(*Request, Share)
	ResponseAfter func(*http.Response, Share)
	ReportHandler func([]*reportor.Result, time.Duration)
}

//Share is a container that is shared in the current transaction,
//you can access the required content.
type Share map[string]interface{}

type Request struct {
	Req       *http.Request
	RoutineNo int
	Index     int
}
