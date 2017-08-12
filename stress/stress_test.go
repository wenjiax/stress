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
)

var task = &Task{
	Number:     100,
	Concurrent: 10,
	// ReportHandler: func(results []*Result, totalTime time.Duration) {},
}

func TestNumber(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}))
	defer ts.Close()

	task.Run(&RequestConfig{
		URLStr: ts.URL,
		Method: "GET",
		Events: &Events{
			ResponseAfter: func(res *http.Response, share Share) {
			},
		},
	})
	if count != 100 {
		t.Error("TestNumber error")
	}
}

func TestDuration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts.Close()

	start := time.Now()
	task.Number = 0
	task.Duration = 2 * time.Second
	task.Run(&RequestConfig{
		URLStr: ts.URL,
		Method: "GET",
	})
	end := time.Now().Sub(start)
	sec := int(end.Seconds())
	if sec != 2 {
		t.Error("TestDuration error")
	}
}

func TestReqBody(t *testing.T) {
	var count int64
	testBody := "Hello Body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		if string(body) == testBody {
			atomic.AddInt64(&count, 1)
		}
	}))
	defer ts.Close()

	task.Number = 100
	task.Duration = 0
	task.Run(&RequestConfig{
		URLStr:  ts.URL,
		Method:  "POST",
		ReqBody: []byte(testBody),
	})
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

	header := make(http.Header, 1)
	header.Set("Content-type", "text/html")
	task.Run(&RequestConfig{
		URLStr: ts.URL,
		Method: "POST",
		Header: header,
	})
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

	events := &Events{
		RequestBefore: func(reqInfo *Request, share Share) {
			reqInfo.Req.Header.Set("test-header", "RequestBefore:Hello")
			reqInfo.Req.Body = ioutil.NopCloser(bytes.NewReader([]byte("RequestBefore:Hello Body")))
		},
		ResponseAfter: func(res *http.Response, share Share) {},
	}
	h := make(http.Header, 1)
	h.Set("test-header", "Hello")
	task.Run(&RequestConfig{
		URLStr:  ts.URL,
		Method:  "POST",
		Header:  h,
		ReqBody: []byte("Hello Body"),
		Events:  events,
	})

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

	events1 := &Events{
		RequestBefore: func(reqInfo *Request, share Share) {},
		ResponseAfter: func(res *http.Response, share Share) {
			body, _ := ioutil.ReadAll(res.Body)
			share["body"] = body
		},
	}
	events2 := &Events{
		RequestBefore: func(reqInfo *Request, share Share) {
			body := share["body"]
			bodyByte := body.([]byte)
			reqInfo.Req.Body = ioutil.NopCloser(bytes.NewReader(bodyByte))
		},
	}
	task.RunTran(&RequestConfig{
		URLStr: ts1.URL,
		Method: "GET",
		Events: events1,
	}, &RequestConfig{
		URLStr: ts2.URL,
		Method: "POST",
		Events: events2,
	})
	if count1 != 100 || count2 != 100 {
		t.Error("TestTran error")
	}
}
