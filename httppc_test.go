// Copyright (C) vislee

package httppc

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestProxyAddrV4(t *testing.T) {
	pp, addr, port := parseAddr("TCP", "127.0.0.1:8080")
	if pp != "TCP4" {
		t.Error("parseAddr protol error")
	}

	if addr != "127.0.0.1" {
		t.Error("parseAddr addr error")
	}

	if port != "8080" {
		t.Error("parseAddr port error")
	}
}

func TestProxyAddrV6(t *testing.T) {
	pp, addr, port := parseAddr("TCP", "[ffff::ffff]:8080")
	if pp != "TCP6" {
		t.Error("parseAddr ipV6 protol error")
	}

	if addr != "ffff::ffff" {
		t.Error("parseAddr ipV6 addr error")
	}

	if port != "8080" {
		t.Error("parseAddr ipV6 port error")
	}
}

func TestProxyProClient(t *testing.T) {
	ch := make(chan string, 1)

	clientAddr := "1.1.1.2"
	serverAddr := "2.2.2.3"
	hostport := "127.0.0.1:8989"
	url := fmt.Sprintf("http://%s/", hostport)

	ln, err := net.Listen("tcp", hostport)
	if err != nil {
		t.Error(err.Error())
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Error(err.Error())
		}

		b := make([]byte, 1024)
		conn.Read(b)
		ch <- string(b)
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nOK"))
		conn.Close()
	}()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Error("new request error", err.Error())
	}
	req.Host = "www.test.com"
	req.Header.Add("X-test-waf", "test-waf")

	pc := NewProxyProClient()
	pc.SetTimeout(3)
	pc.SetProxyProClientIP(clientAddr)
	pc.SetProxyProServerIP(serverAddr)

	resp, err := pc.Do(req)
	if err != nil {
		t.Error("Do error", err.Error())
	}

	if resp.StatusCode != 200 {
		t.Error("resp StatusCode error. ", resp.StatusCode)
	}

	buf := make([]byte, 2)
	n, err := resp.Body.Read(buf)
	if n == 0 && err != nil {
		t.Error(err.Error())
	}

	if n != 2 || string(buf) != "OK" {
		t.Error("body error", n)
	}
	resp.Body.Close()

	bb := <-ch
	if !strings.HasPrefix(bb, fmt.Sprintf("PROXY TCP4 %s %s", clientAddr, serverAddr)) {
		t.Error("proxy protol error.", bb)
	}

	ln.Close()
	close(ch)
}
