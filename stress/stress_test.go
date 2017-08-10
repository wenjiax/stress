package stress

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wenjiax/stress/stress/reportor"
	"github.com/wenjiax/stress/stress/requester"
)

func TestNumber(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}))
	defer ts.Close()
	c := &Config{
		URLStr:     ts.URL,
		Method:     "GET",
		Number:     100,
		Concurrent: 10,
		Events: &requester.Events{
			ReportHandler: func(results []*reportor.Result, total time.Duration) {
				//The result is not handled.
			},
		},
	}
	RunCase(c)
	if count != 100 {
		t.Error("TestNumber error")
	}
}

func TestDuration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts.Close()
	c := &Config{
		URLStr:     ts.URL,
		Method:     "GET",
		Duration:   2 * time.Second,
		Concurrent: 10,
		Events: &requester.Events{
			ReportHandler: func(results []*reportor.Result, total time.Duration) {
				//The result is not handled.
			},
		},
	}
	start := time.Now()
	RunCase(c)
	end := time.Now().Sub(start)
	sec := int(end.Seconds())
	if sec != 2 {
		t.Error("TestDuration error")
	}
}

func TestReqBody(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		if string(body) == "Hello Body" {
			atomic.AddInt64(&count, 1)
		}
	}))
	defer ts.Close()
	c := &Config{
		URLStr:      ts.URL,
		Method:      "POST",
		Number:      100,
		Concurrent:  10,
		RequestBody: "Hello Body",
		Events: &requester.Events{
			ReportHandler: func(results []*reportor.Result, total time.Duration) {
				//The result is not handled.
			},
		},
	}
	RunCase(c)
	if count != 100 {
		t.Error("TestReqBody error")
	}
}

func TestReqHeader(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		if contentType == "text/html" {
			atomic.AddInt64(&count, 1)
		}
	}))
	defer ts.Close()
	header := make(map[string]string, 1)
	header["Content-type"] = "text/html"
	c := &Config{
		URLStr:     ts.URL,
		Method:     "POST",
		Number:     100,
		Concurrent: 10,
		Header:     header,
		Events: &requester.Events{
			ReportHandler: func(results []*reportor.Result, total time.Duration) {
				//The result is not handled.
			},
		},
	}
	RunCase(c)
	if count != 100 {
		t.Error("TestReqHeader error")
	}
}

func TestEvents(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyByte, _ := ioutil.ReadAll(r.Body)
		body := string(bodyByte)
		header := r.Header.Get("test-header")
		if header == "RequestBefore:Hello" && body == "RequestBefore:Hello Body" {
			atomic.AddInt64(&count, 1)
		}
	}))
	defer ts.Close()
	events := &requester.Events{
		RequestBefore: func(reqInfo *requester.Request, share requester.Share) {
			reqInfo.Req.Header.Set("test-header", "RequestBefore:Hello")
			reqInfo.Req.Body = ioutil.NopCloser(bytes.NewReader([]byte("RequestBefore:Hello Body")))
		},
		ResponseAfter: func(res *http.Response, share requester.Share) {

		},
		ReportHandler: func(results []*reportor.Result, total time.Duration) {
			//The result is not handled.
		},
	}
	h := make(map[string]string, 1)
	h["test-header"] = "Hello"
	c := &Config{
		URLStr:      ts.URL,
		Method:      "POST",
		Number:      100,
		Concurrent:  10,
		Events:      events,
		RequestBody: "Hello Body",
		Header:      h,
	}
	RunCase(c)
	if count != 100 {
		t.Error("TestEvents error")
	}
}

func TestTran(t *testing.T) {
	var count1, count2 int64
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count1, int64(1))
		fmt.Fprintf(w, "hello")
	}))
	defer ts1.Close()
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyByte, _ := ioutil.ReadAll(r.Body)
		body := string(bodyByte)
		if body == "hello" {
			atomic.AddInt64(&count2, int64(1))
		}
	}))
	defer ts2.Close()

	events1 := &requester.Events{
		RequestBefore: func(reqInfo *requester.Request, share requester.Share) {
		},
		ResponseAfter: func(res *http.Response, share requester.Share) {
			body, _ := ioutil.ReadAll(res.Body)
			share["body"] = body
		},
		ReportHandler: func(results []*reportor.Result, total time.Duration) {
			//The result is not handled.
		},
	}
	events2 := &requester.Events{
		RequestBefore: func(reqInfo *requester.Request, share requester.Share) {
			body := share["body"]
			bodyByte := body.([]byte)
			reqInfo.Req.Body = ioutil.NopCloser(bytes.NewReader(bodyByte))
		},
	}
	c1 := &Config{
		URLStr:     ts1.URL,
		Method:     "GET",
		Number:     1000,
		Concurrent: 1,
		Events:     events1,
	}
	c2 := &Config{
		URLStr: ts2.URL,
		Method: "POST",
		Events: events2,
	}
	RunTranCase(c1, c2)
	if count1 != 1000 || count2 != 1000 {
		t.Error("TestTran error")
	}
}
