# stress

[![Build Status](https://travis-ci.org/wenjiax/stress.svg?branch=master)](https://travis-ci.org/wenjiax/stress)
[![GoDoc](https://godoc.org/github.com/wenjiax/stress?status.svg)](http://godoc.org/github.com/wenjiax/stress)

stress is an HTTP stress testing tool. Through this tool, you can do a stress test on the HTTP service and get detailed test results. It is inspired by [hey](https://github.com/rakyll/hey).

## Installation

    go get -u github.com/wenjiax/stress

## Features

* **Transactional request support**
* **Support duration and total number of requests**
* **Package reference**
* **Event support**
* **Customizable**
  
## Usage

stress contains two usage, either via the command line or used as a package.

### 1.Use command line.

```
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
```

For example:

```
stress -n 1000 -c 10 -m GET http://localhost:8080
```

Use case: runs a transactional request composed with multiple URL. 

```
stress -n 1000 -c 10 -enable-tran http://localhost:8080,m:post,b:hi http://localhost:8888,m:post,B:./file.txt
```

 ### 2.Use package.

For example:

```
package main

import (
	"fmt"
	stress "github.com/wenjiax/stress/stress"
)

func main() {
	config := &stress.Config{
		URLStr:     "http://localhost:8080/api/test",
		Method:     "GET",
		Duration:   10,         //Continuous request for 10 seconds
		Concurrent: 10,
	}
	err := stress.RunCase(config)
	if err != nil {
		fmt.Println(err)
	}
}
```

Use case: runs a transactional request composed of multiple URL.

```
package main

import (
	"fmt"

	stress "github.com/wenjiax/stress/stress"
)

func main() {
	var configs []*stress.Config
	configs = append(configs, &stress.Config{
		URLStr:     "http://localhost:8080/api/test",
		Method:     "GET",
		Number:     1000,
		Concurrent: 10,
	})
	configs = append(configs, &stress.Config{
		URLStr: "http://localhost:8080/api/GetUserInfo",
		Method: "GET",
	})
	err := stress.RunTranCase(configs...)
	if err != nil {
		fmt.Println(err)
	}
}
```
Add event handling. Make some extra processing before each request, such as setting a different header or request body at a time.
```
    //Share is a container that is shared in the current transaction,
    //you can access the required content.
	events := &requester.Events{
		RequestBefore: func(req *requester.Request, share requester.Share) {
			req.Req.Header.Set("Content-Type", "text/html")
			req.Req.Body = ioutil.NopCloser(bytes.NewReader([]byte("Hello Body")))
			share["name"] = "wenjiax"
		},
		ResponseAfter: func(res *http.Response, share requester.Share) {
			name := share["name"]	//name="wenjiax"
		},
		ReportHandler: func(results []*reportor.Result, total time.Duration) {
			//Custom processing results report.
		},
	}
	config := &stress.Config{
		URLStr:     "http://localhost:8080/api/test",
		Method:     "GET",
		Number:     1000,
		Concurrent: 10,
		Events:     events,
	}
```

## License

stress source code is licensed under the Apache Licence, Version 2.0 (http://www.apache.org/licenses/LICENSE-2.0.html).
