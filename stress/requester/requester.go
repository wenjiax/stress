package requester

import (
	"bytes"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/wenjiax/stress/stress/reportor"
)

type (
	Task struct {
		Number     int
		Concurrent int
		Duration   time.Duration
		Requests   []*RequestInfo
		start      time.Time
		results    []*reportor.Result
		Output     string
		mx         sync.Mutex
	}
	RequestInfo struct {
		Request     *http.Request
		RequestBody []byte
		Events      *Events
		Timeout     int
	}
)

func (t *Task) Run() {
	if t.Duration > 0 {
		t.Number = math.MaxInt32
	} else {
		t.results = make([]*reportor.Result, 0, t.Number)
	}
	t.start = time.Now()
	t.runRequesters()
	t.Finish()

}

func (t *Task) Finish() {
	reportor.NewReport(t.results, t.Output, time.Now().Sub(t.start)).Finalize()
}

func (t *Task) runRequesters() {
	var wg sync.WaitGroup
	wg.Add(t.Concurrent)

	for i := 0; i < t.Concurrent; i++ {
		go func(routineNum int) {
			t.runRequester(t.Number/t.Concurrent, routineNum)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func (t *Task) runRequester(n, no int) {
	client := &http.Client{}
	for i := 0; i < n; i++ {
		if t.Duration > 0 && time.Now().Sub(t.start) >= t.Duration {
			break
		}
		t.sendRequest(client, no, i)
	}
}

func (t *Task) sendRequest(c *http.Client, no, index int) {
	share := make(Share, len(t.Requests))
	results := &reportor.Result{
		Details: make([]*reportor.ResultDetail, len(t.Requests)),
	}
	start := time.Now()
	for i, n := 0, len(t.Requests); i < n; i++ {
		s := time.Now()
		var size int64
		var code int
		var dnsStart, connStart, reqStart, resStart, delayStart, reqBeforeStart, resAfterStart time.Time
		var dnsDuration, connDuration, reqDuration, resDuration, delayDuration, reqBeforeDuration, resAfterDuration time.Duration
		req := cloneRequest(t.Requests[i].Request)
		body := t.Requests[i].RequestBody
		reqBeforeStart = time.Now()
		if t.Requests[i].Events != nil && t.Requests[i].Events.RequestBefore != nil {
			reqInfo := &Request{
				URLStr:    t.Requests[i].Request.URL.String(),
				Method:    t.Requests[i].Request.Method,
				RoutineNo: no,
				Index:     index,
				body:      &body,
				header:    req.Header,
			}
			t.Requests[i].Events.RequestBefore(reqInfo, share)
		}
		reqBeforeDuration = time.Now().Sub(reqBeforeStart)
		if len(body) > 0 {
			req.Body = ioutil.NopCloser(bytes.NewReader(body))
		}
		trace := &httptrace.ClientTrace{
			DNSStart: func(httptrace.DNSStartInfo) {
				dnsStart = time.Now()
			},
			DNSDone: func(httptrace.DNSDoneInfo) {
				dnsDuration = time.Now().Sub(dnsStart)
			},
			GetConn: func(h string) {
				connStart = time.Now()
			},
			GotConn: func(connInfo httptrace.GotConnInfo) {
				connDuration = time.Now().Sub(connStart)
				reqStart = time.Now()
			},
			WroteRequest: func(w httptrace.WroteRequestInfo) {
				reqDuration = time.Now().Sub(reqStart)
				delayStart = time.Now()
			},
			GotFirstResponseByte: func() {
				delayDuration = time.Now().Sub(delayStart)
				resStart = time.Now()
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		c.Timeout = time.Duration(t.Requests[i].Timeout) * time.Second
		res, err := c.Do(req)
		if err == nil {
			size = res.ContentLength
			code = res.StatusCode
			resAfterStart = time.Now()
			if t.Requests[i].Events != nil && t.Requests[i].Events.ResponseAfter != nil {
				t.Requests[i].Events.ResponseAfter(res, share)
			}
			resAfterDuration = time.Now().Sub(resAfterStart)
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}
		t := time.Now()
		resDuration = t.Sub(resStart)
		end := t.Sub(s)
		results.Details[i] = &reportor.ResultDetail{
			URLStr:            req.URL.String(),
			Method:            req.Method,
			Err:               err,
			StatusCode:        code,
			Duration:          end - reqBeforeDuration - resAfterDuration,
			ConnDuration:      connDuration,
			DNSDuration:       dnsDuration,
			ReqDuration:       reqDuration,
			ResDuration:       resDuration,
			DelayDuration:     delayDuration,
			ReqBeforeDuration: reqBeforeDuration,
			ResAfterDuration:  resAfterDuration,
			ContentLength:     size,
		}
	}
	finish := time.Now().Sub(start)
	results.Duration = finish
	t.mx.Lock()
	t.results = append(t.results, results)
	t.mx.Unlock()
}

func cloneRequest(r *http.Request) *http.Request {
	req := new(http.Request)
	*req = *r
	req.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		req.Header[k] = append([]string(nil), s...)
	}
	return req
}
