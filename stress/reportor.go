package stress

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	barChar = "="
)

type (
	// Result is task result.
	Result struct {
		// Details is request details.
		Details []*ResultDetail
		// Duration is the total duration of multiple requests in a transactional request.
		Duration time.Duration
	}
	// ResultDetail is request result details.
	ResultDetail struct {
		// URLStr is the request of URL.
		URLStr string
		// Method is the request of method.
		Method string
		// Err is the error message in the request.
		Err error
		// StatusCode is the status code for the response.
		StatusCode int
		// Duration is request duration.
		Duration time.Duration
		// ConnDuration is connection setup duration.
		ConnDuration time.Duration
		// DNSDuration is dns lookup duration.
		DNSDuration time.Duration
		// ReqDuration is request "write" duration.
		ReqDuration time.Duration
		// ResDuration is response "read" duration.
		ResDuration time.Duration
		// DelayDuration is delay between response and request.
		DelayDuration time.Duration
		// ReqBeforeDuration is function before the request duration.
		ReqBeforeDuration time.Duration
		// ResAfterDuration is function after the response duration.
		ResAfterDuration time.Duration
		// ContentLength is response content length.
		ContentLength int64
	}
	report struct {
		total          time.Duration
		reqBeforeTotal time.Duration
		resAfterTotal  time.Duration
		avgTotal       float64
		fastest        float64
		slowest        float64
		average        float64
		rps            float64

		results    []*Result
		lats       []float64
		details    []*detail
		writers    []io.Writer
		csvWriters []io.Writer
		output     string
	}
	detail struct {
		url            string
		method         string
		avgConn        float64
		avgDNS         float64
		avgReqBefore   float64
		avgReq         float64
		avgResAfter    float64
		avgRes         float64
		avgDelay       float64
		connLats       []float64
		dnsLats        []float64
		reqBeforeLats  []float64
		reqLats        []float64
		resAfterLats   []float64
		resLats        []float64
		delayLats      []float64
		statusCodeDist map[int]int
		errorDist      map[string]int
		sizeTotal      int64
	}
)

func newReport(results []*Result, output string, total time.Duration) *report {
	return &report{
		output:  output,
		results: results,
		total:   total,
	}
}

func (r *report) finalize() {
	for _, result := range r.results {
		r.lats = append(r.lats, result.Duration.Seconds())
		r.avgTotal += result.Duration.Seconds()
		if r.details == nil {
			r.details = make([]*detail, len(result.Details))
		}
		for i, res := range result.Details {
			if r.details[i] == nil {
				r.details[i] = &detail{
					statusCodeDist: make(map[int]int),
					errorDist:      make(map[string]int),
				}

			}
			r.details[i].url = res.URLStr
			r.details[i].method = res.Method
			if res.Err != nil {
				r.details[i].errorDist[res.Err.Error()]++
			} else {
				r.reqBeforeTotal += res.ReqBeforeDuration
				r.resAfterTotal += res.ResAfterDuration
				r.details[i].avgConn += res.ConnDuration.Seconds()
				r.details[i].avgDelay += res.DelayDuration.Seconds()
				r.details[i].avgDNS += res.DNSDuration.Seconds()
				r.details[i].avgReqBefore += res.ReqBeforeDuration.Seconds()
				r.details[i].avgReq += res.ReqDuration.Seconds()
				r.details[i].avgResAfter += res.ResAfterDuration.Seconds()
				r.details[i].avgRes += res.ResDuration.Seconds()
				r.details[i].connLats = append(r.details[i].connLats, res.ConnDuration.Seconds())
				r.details[i].dnsLats = append(r.details[i].dnsLats, res.DNSDuration.Seconds())
				r.details[i].reqBeforeLats = append(r.details[i].reqBeforeLats, res.ReqBeforeDuration.Seconds())
				r.details[i].reqLats = append(r.details[i].reqLats, res.ReqDuration.Seconds())
				r.details[i].delayLats = append(r.details[i].delayLats, res.DelayDuration.Seconds())
				r.details[i].resAfterLats = append(r.details[i].resAfterLats, res.ResAfterDuration.Seconds())
				r.details[i].resLats = append(r.details[i].resLats, res.ResDuration.Seconds())
				r.details[i].statusCodeDist[res.StatusCode]++
				if res.ContentLength > 0 {
					r.details[i].sizeTotal += res.ContentLength
				}
			}
		}
	}
	r.rps = float64(len(r.lats)) / r.total.Seconds()
	r.average = r.avgTotal / float64(len(r.lats))
	for i, n := 0, len(r.details); i < n; i++ {
		r.details[i].avgConn = r.details[i].avgConn / float64(len(r.lats))
		r.details[i].avgDelay = r.details[i].avgDelay / float64(len(r.lats))
		r.details[i].avgDNS = r.details[i].avgDNS / float64(len(r.lats))
		r.details[i].avgReqBefore = r.details[i].avgReqBefore / float64(len(r.lats))
		r.details[i].avgReq = r.details[i].avgReq / float64(len(r.lats))
		r.details[i].avgResAfter = r.details[i].avgResAfter / float64(len(r.lats))
		r.details[i].avgRes = r.details[i].avgRes / float64(len(r.lats))
	}
	r.print()
}

func (r *report) print() {
	r.setWriter()
	if r.output != "" {
		r.printCSV(r.csvWriters...)
	}

	if len(r.lats) > 0 {
		sort.Float64s(r.lats)
		r.fastest = r.lats[0]
		r.slowest = r.lats[len(r.lats)-1]
		r.printf("\nSummary:\n")
		r.printf("  Total:\t\t%4.4f secs\n", r.total.Seconds())
		r.printf("  ReqBeforeTotal:\t%4.4f secs\n", r.reqBeforeTotal.Seconds())
		r.printf("  ResAfterTotal:\t%4.4f secs\n", r.resAfterTotal.Seconds())
		r.printf("  Slowest:\t\t%4.4f secs\n", r.slowest)
		r.printf("  Fastest:\t\t%4.4f secs\n", r.fastest)
		r.printf("  Average:\t\t%4.4f secs\n", r.average)
		r.printf("  Requests/sec:\t\t%4.4f\n", r.rps)
		r.printf("\nDetailed Report:\n")
		for _, detail := range r.details {
			r.printf("\n  URL:  [%s] %s\n", detail.method, detail.url)
			if len(detail.resLats) > 0 {
				r.printSection("DNS+dialup", detail.avgConn, detail.connLats)
				r.printSection("DNS-lookup", detail.avgDNS, detail.dnsLats)
				r.printSection("Request Before", detail.avgReqBefore, detail.reqBeforeLats)
				r.printSection("Request Write", detail.avgReq, detail.reqLats)
				r.printSection("Response Wait", detail.avgDelay, detail.delayLats)
				r.printSection("Response After", detail.avgResAfter, detail.resAfterLats)
				r.printSection("Response Read", detail.avgRes, detail.resLats)
				if detail.sizeTotal > 0 {
					r.printf("\n\tResponse Summary:\n")
					r.printf("\t\tTotal data:\t%d bytes\n", detail.sizeTotal)
					r.printf("\t\tSize/request:\t%d bytes\n", detail.sizeTotal/int64(len(r.lats)))
				}
				r.printStatusCodes(detail.statusCodeDist)
			}
			if len(detail.errorDist) > 0 {
				r.printErrors(detail.errorDist)
			}
		}
		r.printHistogram()
	}
}

func (r *report) setWriter() {
	r.writers = append(r.writers, os.Stdout)
	if r.output != "" {
		fpath := filepath.Join(r.output, "report.txt")
		file, err := os.Create(fpath)
		if err == nil {
			r.writers = append(r.writers, file)
		}
		for _, detail := range r.details {
			_, f := filepath.Split(strings.Replace(strings.Replace(detail.url, ".", "_", -1), ":", "_", -1))
			fpath := filepath.Join(r.output, f)
			file, err := os.Create(fmt.Sprintf("%s.csv", fpath))
			if err == nil {
				r.csvWriters = append(r.csvWriters, file)
			}
		}
	}
}

func (r *report) printCSV(writers ...io.Writer) {
	for _, writer := range writers {
		fmt.Fprintf(writer, "response-time,DNS+dialup,DNS,Request-before,Request-write,Response-delay,Response-after,Response-read\n")
		for _, detail := range r.details {
			for i, val := range detail.reqLats {
				fmt.Fprintf(writer, "%4.4f,%4.4f,%4.4f,%4.4f,%4.4f,%4.4f,%4.4f,%4.4f\n",
					val, detail.connLats[i], detail.dnsLats[i], detail.reqBeforeLats[i], detail.reqLats[i], detail.delayLats[i], detail.resAfterLats[i], detail.resLats[i])
			}
		}
	}
}

func (r *report) printSection(tag string, avg float64, lats []float64) {
	sort.Float64s(lats)
	fastest, slowest := lats[0], lats[len(lats)-1]
	r.printf("\n\t%s:\n", tag)
	r.printf("  \t\tAverage:\t%4.4f secs\n", avg)
	r.printf("  \t\tFastest:\t%4.4f secs\n", fastest)
	r.printf("  \t\tSlowest:\t%4.4f secs\n", slowest)
}

func (r *report) printLatencies() {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.lats)
		if current >= pctls[j] {
			data[j] = r.lats[i]
			j++
		}
	}
	r.printf("\nLatency distribution:\n")
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			r.printf("  %v%% in %4.4f secs\n", pctls[i], data[i])
		}
	}
}

func (r *report) printHistogram() {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (r.slowest - r.fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = r.fastest + bs*float64(i)
	}
	buckets[bc] = r.slowest
	var bi int
	var max int
	for i := 0; i < len(r.lats); {
		if r.lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	r.printf("\nResponse time histogram:\n")
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = (counts[i]*40 + max/2) / max
		}
		r.printf("  %4.3f [%v]\t|%v\n", buckets[i], counts[i], strings.Repeat(barChar, barLen))
	}
}

func (r *report) printStatusCodes(statusCodeDist map[int]int) {
	r.printf("\n\tStatus code distribution:\n")
	for code, num := range statusCodeDist {
		r.printf("\t\t[%d]\t%d responses\n", code, num)
	}
}

func (r *report) printErrors(errorDist map[string]int) {
	r.printf("\n\tError distribution:\n")
	for err, num := range errorDist {
		r.printf("\t\t[%d]\t%s\n", num, err)
	}
}

func (r *report) printf(s string, v ...interface{}) {
	for _, writer := range r.writers {
		fmt.Fprintf(writer, s, v...)
	}
}
