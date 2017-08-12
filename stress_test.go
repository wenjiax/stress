package main

import "testing"

func TestParseValidHeaderFlag(t *testing.T) {
	match, err := parseInputWithRegexp("X-Something: !Y10K:;(He@poverflow?)", headerRegexp)
	if err != nil {
		t.Errorf("A valid header was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "X-Something" || match[2] != "!Y10K:;(He@poverflow?)" {
		t.Errorf("A valid header was not parsed correctly, parsed values: %v %v", match[1], match[2])
	}
}
func TestParseInvalidHeaderFlag(t *testing.T) {
	_, err := parseInputWithRegexp("X|oh|bad-input: badbadbad", headerRegexp)
	if err == nil {
		t.Errorf("An invalid header passed parsing")
	}
}

func TestParseValidMethodFlag(t *testing.T) {
	match, err := parseInputWithRegexp("http://127.0.0.1:8080,m:GET", methodsRegexp)
	if err != nil {
		t.Errorf("A valid method was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "GET" {
		t.Errorf("A valid method was not parsed correctly, parsed values: %v", match[1])
	}
}

func TestParseValidBodyFlag(t *testing.T) {
	match, err := parseInputWithRegexp("http://127.0.0.1:8080,b:hello body", bodyRegexp)
	if err != nil {
		t.Errorf("A valid body was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "hello body" {
		t.Errorf("A valid body was not parsed correctly, parsed values: %v", match[1])
	}
}

func TestParseValidBodyFileFlag(t *testing.T) {
	match, err := parseInputWithRegexp("http://127.0.0.1:8080,B:./bodyFile.txt", bodyFileRegexp)
	if err != nil {
		t.Errorf("A valid bodyFile was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "./bodyFile.txt" {
		t.Errorf("A valid bodyFile was not parsed correctly, parsed values: %v", match[1])
	}
}

func TestParseValidProxyAddrFlag(t *testing.T) {
	match, err := parseInputWithRegexp("http://127.0.0.1:8080,x:http://127.0.0.1:8888", proxyAddrRegexp)
	if err != nil {
		t.Errorf("A valid ProxyAddr was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "http://127.0.0.1:8888" {
		t.Errorf("A valid ProxyAddr was not parsed correctly, parsed values: %v", match[1])
	}
}

func TestParseValidThinkTimeFlag(t *testing.T) {
	match, err := parseInputWithRegexp("http://127.0.0.1:8080,thinkTime:2", thinkTimeRegexp)
	if err != nil {
		t.Errorf("A valid ThinkTime was not parsed correctly: %v", err.Error())
		return
	}
	if match[1] != "2" {
		t.Errorf("A valid ThinkTime was not parsed correctly, parsed values: %v", match[1])
	}
}

func TestParseInvalidThinkTimeFlag(t *testing.T) {
	_, err := parseInputWithRegexp("http://127.0.0.1:8080,thinkTime:abc", thinkTimeRegexp)
	if err == nil {
		t.Errorf("An invalid ThinkTime passed parsing")
	}
}
