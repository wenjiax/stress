# stress

[![Build Status](https://travis-ci.org/wenjiax/stress.svg?branch=master)](https://travis-ci.org/wenjiax/stress)
[![GoDoc](https://godoc.org/github.com/wenjiax/stress?status.svg)](http://godoc.org/github.com/wenjiax/stress)

stress is an HTTP stress testing tool. Through this tool, you can do a stress test on the HTTP service and get detailed test results. It is inspired by [hey](https://github.com/rakyll/hey).

## Installation

    go get -u github.com/wenjiax/stress
 
## Features
 
* **Transactional task support**
* **Support duration and total number of requests**
* **Support package reference**
* **Support custom event**
* **Support customizable**
  
## Usage

stress contains two usage, either via the command line or used as a package.

### 1.Use command line.

```
Usage: stress [options...] <url> || stress [options...] -enable-tran <urls...>

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
```

For example: run a task.

```
stress -n 1000 -c 10 -m GET http://localhost:8080
```

Use task: run a transactional request composed of multiple URL.

```
stress -n 1000 -c 10 -enable-tran http://localhost:8080,m:post,b:hi,x:http://127.0.0.1:8888 http://localhost:8888,m:post,B:/home/file.txt,thinkTime:2 
```

 ### 2.Use package.

For example: run a task.

```
package main

import (
	"fmt"

	stress "github.com/wenjiax/stress/stress"
)

func main() {
	task := &stress.Task{
		Duration:   10, //Continuous request for 10 seconds
		Concurrent: 10,
		ReportHandler: func(results []*stress.Result, totalTime time.Duration) {
			//Processing result reporting function.
			//If the function is passed in, the incoming function is used to process the report,
			//otherwise the default function is used to process the report.
		},
	}
	err := task.Run(&stress.RequestConfig{
		URLStr: "http://localhost:8080/api/test",
		Method: "GET",
	})
	if err != nil {
		fmt.Println(err)
	}
}

```

For example: run a transactional request composed of multiple URL.

```
package main

import (
	"fmt"

	stress "github.com/wenjiax/stress/stress"
)

func main() {
	task := &stress.Task{
		Number:     1000,
		Concurrent: 10,
	}
	var configs []*stress.RequestConfig
	configs = append(configs, &stress.RequestConfig{
		URLStr: "http://localhost:8080/api/test",
		Method: "GET",
	})
	configs = append(configs, &stress.RequestConfig{
		URLStr: "http://localhost:8080/api/hello",
		Method: "POST",
	})
	err := task.RunTran(configs...)
	if err != nil {
		fmt.Println(err)
	}
}

```
Add event handling. Make some extra processing before each request, such as setting a different header or request body at a time.
```
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	stress "github.com/wenjiax/stress/stress"
)

func main() {
	task := &stress.Task{
		Number:     1000,
		Concurrent: 10,
	}
	events := &stress.Events{
		//Share is a container that is shared in the current transaction,
		//you can access the required content.
		RequestBefore: func(req *stress.Request, share stress.Share) {
			req.Req.Header.Set("Content-Type", "text/html")
			req.Req.Body = ioutil.NopCloser(bytes.NewReader([]byte("Hello Body")))
			share["name"] = "wenjiax"
		},
		ResponseAfter: func(res *http.Response, share stress.Share) {
			name := share["name"] //name="wenjiax"
			fmt.Println(name)
		},
	}
	err := task.Run(&stress.RequestConfig{
		URLStr: "http://localhost:8080/api/test",
		Method: "GET",
		Events: events,
	})
	if err != nil {
		fmt.Println(err)
	}
}

```

## License

stress source code is licensed under the Apache Licence, Version 2.0 (http://www.apache.org/licenses/LICENSE-2.0.html).
