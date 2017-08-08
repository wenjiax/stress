package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	lbstress "github.com/wenjiax/stress/stress"
)

var (
	m        = flag.String("m", "GET", "")
	headers  = flag.String("h", "", "")
	body     = flag.String("b", "", "")
	bodyFile = flag.String("B", "", "")

	output = flag.String("o", "", "")

	n = flag.Int("n", 0, "")
	c = flag.Int("c", 0, "")
	t = flag.Int("t", 20, "")
	d = flag.Int("d", 0, "")

	enableTran = flag.Bool("enable-tran", false, "")
)

const (
	methodsRegexp  = `m:([a-zA-Z]+),*`
	bodyRegexp     = `b:([^,]+),*`
	bodyFileRegexp = `B:([^,]+),*`
	headerRegexp   = `^([\w-]+):\s*(.+)`
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
      -h "Accept: text/html;Content-Type: application/xml".
  -m  HTTP method, any of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -t  Timeout for each request in seconds. Default value is 20, 
      use 0 for infinite.
  -b  HTTP request body.
  -B  HTTP request body from file. For example:
      /home/user/file.txt or ./file.txt.
  
  -enable-tran  Enable transactional requests. Multiple urls 
                form a transactional requests. 
                For example: "stress [options...] -enable-tran 
                http://localhost:8080,m:post,b:hi 
                http://localhost:8888,m:post,B:/home/file.txt [urls...]".
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()
	if flag.NArg() <= 0 {
		usageAndExit("")
	}
	header := make(map[string]string)
	hs := strings.Split(*headers, ";")
	for _, h := range hs {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		header[match[1]] = match[2]
	}
	if !*enableTran {
		var bodyAll string
		if *body != "" {
			bodyAll = *body
		}
		if *bodyFile != "" {
			content, err := ioutil.ReadFile(*bodyFile)
			if err != nil {
				errAndExit(err.Error())
			}
			bodyAll = string(content)
		}
		err := lbstress.RunCase(&lbstress.Config{
			URLStr:      flag.Args()[0],
			Method:      *m,
			Number:      *n,
			Concurrent:  *c,
			Duration:    time.Duration(*d) * time.Second,
			Timeout:     *t,
			Output:      *output,
			RequestBody: bodyAll,
			Header:      header,
		})
		if err != nil {
			errAndExit(err.Error())
		}
		return
	}
	var configs []*lbstress.Config
	for i, len := 0, flag.NArg(); i < len; i++ {
		argstr := flag.Args()[i]
		url := strings.Split(argstr, ",")[0]
		methodMatch, err := parseInputWithRegexp(argstr, methodsRegexp)
		if err != nil {
			errAndExit(err.Error())
		}
		bodyMatch, _ := parseInputWithRegexp(argstr, bodyRegexp)
		bodyFileMatch, _ := parseInputWithRegexp(argstr, bodyFileRegexp)
		var bodyAll string
		if bodyMatch != nil {
			bodyAll = bodyMatch[1]
		}
		if bodyFileMatch != nil {
			content, err := ioutil.ReadFile(bodyFileMatch[1])
			if err != nil {
				errAndExit(err.Error())
			}
			bodyAll = string(content)
		}
		configs = append(configs, &lbstress.Config{
			URLStr:      url,
			Method:      methodMatch[1],
			Number:      *n,
			Concurrent:  *c,
			Duration:    time.Duration(*d) * time.Second,
			Timeout:     *t,
			Output:      *output,
			RequestBody: bodyAll,
			Header:      header,
		})
	}
	err := lbstress.RunTranCase(configs...)
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
	fmt.Fprintf(os.Stderr, msg)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
