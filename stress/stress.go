package stress

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/wenjiax/stress/stress/requester"
)

type Config struct {
	URLStr      string
	Method      string
	Number      int
	Concurrent  int
	Duration    time.Duration
	RequestBody string
	Header      map[string]string
	Events      *requester.Events
	Timeout     int
	Output      string
	request     *http.Request
}

//RunCase Run a test case.
func RunCase(config *Config) error {
	if err := checkInputParams(config); err != nil {
		return err
	}
	if config.URLStr == "" || config.Method == "" {
		return errors.New("URLStr and Method cannot be empty")
	}
	method := strings.ToUpper(config.Method)
	req, err := http.NewRequest(method, config.URLStr, nil)
	if err != nil {
		return err
	}
	if config.Header != nil {
		for k, v := range config.Header {
			req.Header.Set(k, v)
		}
	}
	config.request = req
	run(config)
	return nil
}

//RunTranCase Run a test case that contains transactions.
func RunTranCase(configs ...*Config) error {
	if configs == nil {
		return errors.New("configs cannot be nil")
	}
	if err := checkInputParams(configs[0]); err != nil {
		return err
	}
	for i, n := 0, len(configs); i < n; i++ {
		if configs[i].URLStr == "" || configs[i].Method == "" {
			return errors.New("URLStr and Method cannot be empty")
		}
		method := strings.ToUpper(configs[i].Method)
		req, err := http.NewRequest(method, configs[i].URLStr, nil)
		if err != nil {
			return err
		}
		if configs[i].Header != nil {
			for k, v := range configs[i].Header {
				req.Header.Set(k, v)
			}
		}
		configs[i].request = req
	}
	run(configs...)
	return nil
}

func checkInputParams(config *Config) error {
	if config == nil {
		return errors.New("configs cannot be nil")
	}
	if config.Number <= 0 && config.Duration <= 0 {
		return errors.New("Number or Duration cannot be smaller than 1")
	}
	if config.Number > 0 && config.Duration > 0 {
		return errors.New("Number and Duration only set one")
	}
	if config.Concurrent <= 0 {
		return errors.New("Concurrent cannot be smaller than 1")
	}
	if config.Number > 0 && config.Number < config.Concurrent {
		return errors.New("Number cannot be less than Concurrent")
	}
	if config.Number > 0 && config.Number%config.Concurrent != 0 {
		return errors.New("Number must be an integer multiple of Concurrent")
	}
	if config.Output != "" {
		err := os.MkdirAll(config.Output, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func run(configs ...*Config) {
	var reqs []*requester.RequestInfo
	for i, n := 0, len(configs); i < n; i++ {
		timeout := 20
		if configs[i].Timeout > 0 {
			timeout = configs[i].Timeout
		}
		req := &requester.RequestInfo{
			Request:     configs[i].request,
			RequestBody: []byte(configs[i].RequestBody),
			Events:      configs[i].Events,
			Timeout:     timeout,
		}
		reqs = append(reqs, req)
	}
	work := &requester.Task{
		Requests:   reqs,
		Number:     configs[0].Number,
		Concurrent: configs[0].Concurrent,
		Duration:   configs[0].Duration,
		Output:     configs[0].Output,
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		work.Finish()
		os.Exit(1)
	}()
	work.Run()
}
