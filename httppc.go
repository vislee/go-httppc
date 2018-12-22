// Copyright (C) vislee

package httppc

import (
	"bytes"
	"context"
	// "log"
	"net"
	"net/http"
	"strings"
	"time"
)

type proxyProClient struct {
	http.Client
}

type proxyProDialer struct {
	net.Dialer
	cliAddr string
	serAddr string
}

func (self *proxyProDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := self.Dialer.DialContext(ctx, network, address)

	if err != nil {
		// log.Println(err.Error())
		return conn, err
	}

	var (
		protol  string
		cliAddr string
		cliPort string
		serAddr string
		serPort string
	)

	lAddr := conn.LocalAddr()
	raddr := conn.RemoteAddr()

	protol, serAddr, serPort = parseAddr(raddr.Network(), raddr.String())
	_, cliAddr, cliPort = parseAddr(lAddr.Network(), lAddr.String())

	if self.serAddr != "" {
		serAddr = self.serAddr

		if addrs := strings.Split(self.serAddr, ":"); len(addrs) == 2 {
			serPort = addrs[1]
			serAddr = addrs[0]
		}
	}

	if self.cliAddr != "" {
		cliAddr = self.cliAddr
	}

	ppBuff := bytes.NewBuffer([]byte("PROXY "))

	ppBuff.WriteString(protol)
	ppBuff.WriteString(" ")

	ppBuff.WriteString(cliAddr)
	ppBuff.WriteString(" ")

	ppBuff.WriteString(serAddr)
	ppBuff.WriteString(" ")

	ppBuff.WriteString(cliPort)
	ppBuff.WriteString(" ")

	ppBuff.WriteString(serPort)
	ppBuff.WriteString("\r\n")

	_, err = conn.Write(ppBuff.Bytes())
	if err != nil {
		conn.Close()
	}

	return conn, err
}

var ppDialer = &proxyProDialer{
	Dialer: net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second, DualStack: true},
}

var ppTransport http.RoundTripper = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           ppDialer.DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

func NewProxyProClient() *proxyProClient {
	pc := &proxyProClient{
		http.Client{
			Transport: ppTransport,
			Timeout:   90 * time.Second,
		},
	}
	return pc
}

func (self *proxyProClient) NotFollowRedirects() {
	self.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
}

func (self *proxyProClient) SetTimeout(d time.Duration) {
	self.Client.Timeout = d * time.Second
}

func (self *proxyProClient) SetProxyProClientIP(remoteAddr string) {
	ppDialer.cliAddr = remoteAddr
}

func (self *proxyProClient) SetProxyProServerIP(serAddr string) {
	ppDialer.serAddr = serAddr
}

func parseAddr(net, addr string) (string, string, string) {
	if net == "" {
		return "UNKNOWN", "", ""
	}

	// log.Println(addr)

	colon := strings.IndexByte(addr, ':')
	if colon == -1 {
		return strings.ToUpper(net) + "4", addr, "80"
	}

	if i := strings.Index(addr, "]:"); i != -1 {
		return strings.ToUpper(net) + "6", addr[1:i], addr[i+len("]:"):]
	}
	if strings.Contains(addr, "]") {
		return strings.ToUpper(net) + "6", addr[1 : len(addr)-2], "80"
	}

	return strings.ToUpper(net) + "4", addr[:colon], addr[colon+len(":"):]
}
