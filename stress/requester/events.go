package requester

import (
	"github.com/wenjiax/stress/stress/reportor"
	"net/http"
)

type Events struct {
	RequestBefore func(*Request, Share)
	ResponseAfter func(*http.Response, Share)
	ReportHandler func([]*reportor.Result)
}

//Share is a container that is shared in the current transaction,
//you can access the required content.
type Share map[string]interface{}

type Request struct {
	URLStr    string
	Method    string
	RoutineNo int
	Index     int
	body      *[]byte
	header    http.Header
}

func (r *Request) SetBody(bodyStr string) {
	*r.body = []byte(bodyStr)
}

func (r *Request) SetHeader(key, value string) {
	_, ok := (r.header)[key]
	if ok {
		r.header.Add(key, value)
	} else {
		r.header.Set(key, value)
	}
}
