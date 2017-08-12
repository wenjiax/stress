package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	gurl "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	lbstress "github.com/wenjiax/stress/stress"
)

var (
	m = flag.String("m", "GET", "")
	// headers  = flag.String("h", "", "")
	body     = flag.String("b", "", "")
	bodyFile = flag.String("B", "", "")

	output    = flag.String("o", "", "")
	proxyAddr = flag.String("x", "", "")

	n         = flag.Int("n", 0, "")
	c         = flag.Int("c", 0, "")
	t         = flag.Int("t", 20, "")
	d         = flag.Int("d", 0, "")
	thinkTime = flag.Int("think-time", 0, "")

	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepalive   = flag.Bool("disable-keepalive", false, "")
	disableRedirects   = flag.Bool("disable-redirects", false, "")
	enableTran         = flag.Bool("enable-tran", false, "")
)

const (
	headerRegexp = `^([\w-]+):\s*(.+)`

	methodsRegexp   = `m:([a-zA-Z]+),*`
	bodyRegexp      = `b:([^,]+),*`
	bodyFileRegexp  = `B:([^,]+),*`
	proxyAddrRegexp = `x:([^,]+),*`
	thinkTimeRegexp = `thinkTime:([\d]+),*`
)

var usage = `Usage: stress [options...] <url> || stress [options...] -enable-tran <urls...>

Options:
  -n  Number of requests to run. Default value is 0.
  -c  Number of requests to run concurrently. 
      Total number of requests cannot smaller than the concurrency level. 
      Default value is 0.
  -d  Duration of requests to run. Default value is 0 sec.
  -o  Output file path. For example: /home/user or ./files.
  
  -h  Custom HTTP header. For example: 
      -h "Accept: text/html" -h "Content-Type: application/xml".
  -m  HTTP method, any of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -t  Timeout for each request in seconds. Default value is 20, 
      use 0 for infinite.
  -b  HTTP request body.
  -B  HTTP request body from file. For example:
      /home/user/file.txt or ./file.txt.
  -x  HTTP Proxy address as host:port.

  -think-time           Time to think after request. Default value is 0 sec.
  -disable-compression  Disable compression.
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                    	connections between different HTTP requests.
  -disable-redirects    Disable following of HTTP redirects.
  -enable-tran          Enable transactional requests. Multiple urls 
                        form a transactional requests. 
                        For example: "stress [options...] -enable-tran 
                        http://localhost:8080,m:post,b:hi,x:http://127.0.0.1:8888 
                        http://localhost:8888,m:post,B:/home/file.txt,thinkTime:2 
                        [urls...]".
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	var hs headerSlice
	flag.Var(&hs, "h", "")

	flag.Parse()
	if flag.NArg() <= 0 {
		usageAndExit("")
	}
	//Parsing global request header.
	header := make(http.Header)
	// hs := strings.Split(*headers, ";")
	for _, h := range hs {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		header.Set(match[1], match[2])
	}
	//Parsing global request proxyAddr.
	var proxyURL *gurl.URL
	if *proxyAddr != "" {
		var err error
		proxyURL, err = gurl.Parse(*proxyAddr)
		if err != nil {
			usageAndExit(err.Error())
		}
	}
	//Set parameters and global configuration.
	task := &lbstress.Task{
		Number:             *n,
		Concurrent:         *c,
		Duration:           time.Duration(*d) * time.Second,
		Output:             *output,
		Timeout:            *t,
		ThinkTime:          *thinkTime,
		ProxyAddr:          proxyURL,
		DisableCompression: *disableCompression,
		DisableKeepAlives:  *disableKeepalive,
		DisableRedirects:   *disableRedirects,
	}
	if *enableTran {
		runTran(task, header)
	} else {
		run(task, header)
	}

}

func run(task *lbstress.Task, header http.Header) {
	//Parsing request body.
	var bodyAll []byte
	if *body != "" {
		bodyAll = []byte(*body)
	}
	if *bodyFile != "" {
		content, err := ioutil.ReadFile(*bodyFile)
		if err != nil {
			errAndExit(err.Error())
		}
		bodyAll = content
	}
	//Run task.
	err := task.Run(&lbstress.RequestConfig{
		URLStr:  flag.Args()[0],
		Method:  *m,
		ReqBody: bodyAll,
		Header:  header,
	})
	if err != nil {
		errAndExit(err.Error())
	}
}

func runTran(task *lbstress.Task, header http.Header) {
	var configs []*lbstress.RequestConfig
	for i, len := 0, flag.NArg(); i < len; i++ {
		argstr := flag.Args()[i]
		url := strings.Split(argstr, ",")[0]
		//Parsing request method.
		methodMatch, err := parseInputWithRegexp(argstr, methodsRegexp)
		if err != nil {
			errAndExit(err.Error())
		}
		//Parsing request body.
		bodyMatch, _ := parseInputWithRegexp(argstr, bodyRegexp)
		bodyFileMatch, _ := parseInputWithRegexp(argstr, bodyFileRegexp)
		var bodyAll []byte
		if bodyMatch != nil {
			bodyAll = []byte(bodyMatch[1])
		}
		if bodyFileMatch != nil {
			content, err := ioutil.ReadFile(bodyFileMatch[1])
			if err != nil {
				errAndExit(err.Error())
			}
			bodyAll = content
		}
		//Parsing request proxyAddr.
		proxyAddrMatch, _ := parseInputWithRegexp(argstr, proxyAddrRegexp)
		var proxyURL *gurl.URL
		if proxyAddrMatch != nil {
			var err error
			proxyURL, err = gurl.Parse(*proxyAddr)
			if err != nil {
				usageAndExit(err.Error())
			}
		}
		//Parsing request thinkTime.
		thinkTime := 0
		thinkTimeMatch, _ := parseInputWithRegexp(argstr, thinkTimeRegexp)
		if thinkTimeMatch != nil {
			thinkTime, _ = strconv.Atoi(thinkTimeMatch[1])
		}
		configs = append(configs, &lbstress.RequestConfig{
			URLStr:    url,
			Method:    methodMatch[1],
			ReqBody:   bodyAll,
			Header:    header,
			ProxyAddr: proxyURL,
			ThinkTime: thinkTime,
		})
	}
	//Run transactional task.
	err := task.RunTran(configs...)
	if err != nil {
		errAndExit(err.Error())
	}
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
	}
	return matches, nil
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, "Error:%s\n", msg)
	os.Exit(1)
}

type headerSlice []string

func (h *headerSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *headerSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}
