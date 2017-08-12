package stress

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type (
	//Task contains the stress test configuration and request configuration.
	Task struct {
		//Nuber is the total number of requests to send.
		Number int
		//Concurrent is the concurrent number of requests.
		Concurrent int
		//Duration is the duration of requests.
		Duration time.Duration
		//Output is the report output directory.
		//The output contains the summary information file and the CSV file for each request.
		Output string
		//Processing result reporting function.
		//If the function is passed in, the incoming function is used to process the report,
		//otherwise the default function is used to process the report.
		ReportHandler func(results []*Result, totalTime time.Duration)

		//Global configuration, if the configuration is not specified in RequestConfig,
		//use the settings global configuration.
		//Timeout is the timeout of request in seconds.
		Timeout int
		//ThinkTime is the think time of request in seconds.
		ThinkTime int
		// ProxyAddr is the address of HTTP proxy server in the format on "host:port".
		ProxyAddr *url.URL
		//HTTP Host header
		Host string
		//H2 is an option to make HTTP/2 requests.
		H2 bool
		//DisableCompression is an option to disable compression in response.
		DisableCompression bool
		//DisableKeepAlives is an option to prevents re-use of TCP connections between different HTTP requests.
		DisableKeepAlives bool
		//DisableRedirects is an option to prevent the following of HTTP redirects.
		DisableRedirects bool

		//The total think time required for all requests.
		thinkDuration time.Duration
		reqConfigs    []*RequestConfig
		start         time.Time
		results       []*Result
		mx            sync.Mutex
	}
	//RequestConfig is the request of configuration.
	RequestConfig struct {
		//URLStr is the request of URL.
		URLStr string
		//Method is the request of method.
		Method string
		//ReqBody is the request of body.
		ReqBody []byte
		//Header is the request of header.
		Header http.Header
		//Events is the custom event in the request.
		//Contains the function before the request and the function after the response.
		Events *Events

		//Timeout is the timeout of request in seconds.
		Timeout int
		//ThinkTime is the think time of request in seconds.
		ThinkTime int
		// ProxyAddr is the address of HTTP proxy server in the format on "host:port".
		ProxyAddr *url.URL
		//HTTP Host header
		Host string
		//H2 is an option to make HTTP/2 requests.
		H2 bool
		//DisableCompression is an option to disable compression in response.
		DisableCompression bool
		//DisableKeepAlives is an option to prevents re-use of TCP connections between different HTTP requests.
		DisableKeepAlives bool
		//DisableRedirects is an option to prevent the following of HTTP redirects.
		DisableRedirects bool

		request *http.Request
		client  *http.Client
	}
)

//Run is run a task.
func (t *Task) Run(config *RequestConfig) error {
	t.reqConfigs = append([]*RequestConfig(nil), config)
	return t.run()
}

//RunTran is run a transactional task.
func (t *Task) RunTran(configs ...*RequestConfig) error {
	t.reqConfigs = append([]*RequestConfig(nil), configs...)
	return t.run()
}

func (t *Task) run() error {
	if err := t.checkAndInitConfigs(); err != nil {
		return err
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		t.finish()
		os.Exit(1)
	}()
	t.start = time.Now()
	t.runRequesters()
	t.finish()
	return nil
}

func (t *Task) finish() {
	if t.Number < 0 && t.ReportHandler == nil {
		return
	}
	total := time.Now().Sub(t.start) - t.thinkDuration
	if t.ReportHandler != nil {
		t.ReportHandler(t.results, total)
	} else {
		newReport(t.results, t.Output, total).finalize()
	}
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

func (t *Task) runRequester(num, no int) {
	i := 0
	if t.Duration > 0 || t.Number < 0 {
		for {
			if t.Duration > 0 && time.Now().Sub(t.start) >= t.Duration {
				break
			}
			t.sendRequest(no, i)
			i++
		}
		return
	}
	for ; i < num; i++ {
		t.sendRequest(no, i)
	}
}

func (t *Task) sendRequest(no, index int) {
	//init share and results.
	len := len(t.reqConfigs)
	share := make(Share, len)
	results := &Result{
		Details: make([]*ResultDetail, len),
	}
	tranStart := time.Now()
	var thinkDuration time.Duration
	for i, reqConfig := range t.reqConfigs {
		start := time.Now()
		var size int64
		var code int
		var dnsStart, connStart, reqStart, resStart, delayStart, reqBeforeStart, resAfterStart time.Time
		var dnsDuration, connDuration, reqDuration, resDuration, delayDuration, reqBeforeDuration, resAfterDuration time.Duration
		req := cloneRequest(reqConfig.request, reqConfig.ReqBody)
		req.Host = reqConfig.Host
		//Handle custom event: function before the request.
		reqBeforeStart = time.Now()
		if reqConfig.Events != nil && reqConfig.Events.RequestBefore != nil {
			reqInfo := &Request{
				GoRoutineNo: no,
				Index:       index,
				Req:         req,
			}
			reqConfig.Events.RequestBefore(reqInfo, share)
		}
		reqBeforeDuration = time.Now().Sub(reqBeforeStart)
		//Create http.Client.
		client := reqConfig.client
		if client == nil {
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				DisableCompression: reqConfig.DisableCompression,
				DisableKeepAlives:  reqConfig.DisableKeepAlives,
				Proxy:              http.ProxyURL(reqConfig.ProxyAddr),
			}
			if reqConfig.H2 {
				http2.ConfigureTransport(transport)
			} else {
				transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
			}
			client = &http.Client{
				Transport: transport,
				Timeout:   time.Duration(reqConfig.Timeout) * time.Second,
			}
			if reqConfig.DisableRedirects {
				client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}
			}
			t.mx.Lock()
			t.reqConfigs[i].client = client
			t.mx.Unlock()
		}
		//Create httptrace.
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
		res, err := client.Do(req)
		if err == nil {
			size = res.ContentLength
			code = res.StatusCode
			//Handle custom event: function after the response.
			resAfterStart = time.Now()
			if reqConfig.Events != nil && reqConfig.Events.ResponseAfter != nil {
				reqConfig.Events.ResponseAfter(res, share)
			}
			resAfterDuration = time.Now().Sub(resAfterStart)
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}
		nowTime := time.Now()
		resDuration = nowTime.Sub(resStart)
		end := nowTime.Sub(start)
		results.Details[i] = &ResultDetail{
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
		//Handle think time.
		thinktime := time.Duration(reqConfig.ThinkTime) * time.Second
		time.Sleep(thinktime)
		thinkDuration += thinktime
		t.thinkDuration += thinktime
	}
	finish := time.Now().Sub(tranStart)
	results.Duration = finish - thinkDuration
	//Save request result.
	if t.Number < 0 && t.ReportHandler == nil {
		return
	}
	t.mx.Lock()
	t.results = append(t.results, results)
	t.mx.Unlock()
}

func cloneRequest(r *http.Request, body []byte) *http.Request {
	req := new(http.Request)
	*req = *r
	req.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		req.Header[k] = append([]string(nil), s...)
	}
	if len(body) > 0 {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	return req
}

func (t *Task) checkAndInitConfigs() error {
	if t.Number == 0 && t.Duration <= 0 {
		return errors.New("Number or Duration cannot be smaller than 1")
	}
	if t.Number != 0 && t.Duration > 0 {
		return errors.New("Number and Duration only set one")
	}
	if t.Concurrent <= 0 {
		return errors.New("Concurrent cannot be smaller than 1")
	}
	if t.Number > 0 && t.Number < t.Concurrent {
		return errors.New("Number cannot be less than Concurrent")
	}
	if t.Number > 0 && t.Number%t.Concurrent != 0 {
		return errors.New("Number must be an integer multiple of Concurrent")
	}
	if t.Output != "" {
		err := os.MkdirAll(t.Output, 0777)
		if err != nil {
			return err
		}
	}
	if t.Duration <= 0 && t.Number > 0 {
		t.results = make([]*Result, 0, t.Number)
	}
	for i, n := 0, len(t.reqConfigs); i < n; i++ {
		if t.reqConfigs[i] == nil {
			return errors.New("RequestConfig cannot be nil")
		}
		if t.reqConfigs[i].URLStr == "" || t.reqConfigs[i].Method == "" {
			return errors.New("URLStr and Method cannot be empty")
		}
		if t.Timeout > 0 && t.reqConfigs[i].Timeout <= 0 {
			t.reqConfigs[i].Timeout = t.Timeout
		}
		if t.ThinkTime > 0 && t.reqConfigs[i].ThinkTime <= 0 {
			t.reqConfigs[i].ThinkTime = t.ThinkTime
		}
		if t.Host != "" && t.reqConfigs[i].Host == "" {
			t.reqConfigs[i].Host = t.Host
		}
		if t.ProxyAddr != nil && t.reqConfigs[i].ProxyAddr == nil {
			t.reqConfigs[i].ProxyAddr = t.ProxyAddr
		}
		if t.DisableCompression && !t.reqConfigs[i].DisableCompression {
			t.reqConfigs[i].DisableCompression = true
		}
		if t.DisableKeepAlives && !t.reqConfigs[i].DisableKeepAlives {
			t.reqConfigs[i].DisableKeepAlives = true
		}
		if t.DisableRedirects && !t.reqConfigs[i].DisableRedirects {
			t.reqConfigs[i].DisableRedirects = true
		}
		t.reqConfigs[i].Method = strings.ToUpper(t.reqConfigs[i].Method)
		req, err := http.NewRequest(t.reqConfigs[i].Method, t.reqConfigs[i].URLStr, nil)
		if err != nil {
			return err
		}
		if t.reqConfigs[i].Header != nil {
			req.Header = t.reqConfigs[i].Header
		}
		t.reqConfigs[i].request = req
	}

	return nil
}
