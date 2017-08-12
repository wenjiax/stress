package stress

import (
	"net/http"
)

//Events is the custom event in the request.
type Events struct {
	//RequestBefore is function before the request.
	RequestBefore func(req *Request, share Share)
	//ResponseAfter is function after the response.
	ResponseAfter func(res *http.Response, share Share)
}

//Share is a container that is shared in the current transaction,
//you can access the required content.
type Share map[string]interface{}

//Request is request info.
type Request struct {
	//Req is http.Request.
	Req *http.Request
	//GoRoutineNo is the current executed goroutine serial number.
	GoRoutineNo int
	//Index is current executed index.
	Index int
}
