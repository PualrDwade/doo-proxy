package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	"github.com/siddontang/go/log"
)

var credential = flag.String("credential", "", "set set  for proxy server")

func main() {
	flag.Parse()
	if credential == nil || *credential == "" {
		flag.Usage()
		return
	}
	proxy := NewDooProxy("localhost:5050", *credential)
	proxy.Start()
}

// NewDooProxy return proxy server instance
func NewDooProxy(addr, credential string) DooProxy {
	// todo validate addr
	return &dooProxy{
		addr:       addr,
		credential: credential,
	}
}

// DooProxy http/https proxy server
type DooProxy interface {
	Start()
}

type dooProxy struct {
	listener   net.Listener
	addr       string
	credential string
}

// Start start a proxy server
func (proxy *dooProxy) Start() {
	var err error
	proxy.listener, err = net.Listen("tcp", proxy.addr)
	if err != nil {
		panic(err.Error())
	}

	if proxy.credential != "" {
		log.Infof("use credential %v for proxy server", proxy.credential)
	}

	log.Infof("listen in %s, wating client to connet...", proxy.addr)
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Errorf("accept client faild : %v ", err)
		}
		go proxy.handleConn(conn)
	}
}

func (proxy *dooProxy) handleConn(clientConn net.Conn) {
	// handle cient conn step:
	// 1. extract base information from cient request
	// 2. connect remote host and hold connction
	// 3. begin transport data from client and remote server
	defer clientConn.Close()
	rawHTTPRequestHeader, remote, credential, isHTTPS, err := proxy.extractTunnelInfo(clientConn)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if proxy.auth(clientConn, credential) == false {
		log.Error("Auth fail: " + credential)
		return
	}

	log.Info("connecting to " + remote)
	remoteConn, err := net.Dial("tcp", remote)
	defer remoteConn.Close()
	if err != nil {
		log.Error(err.Error())
		return
	}
	if isHTTPS {
		// is https, sent 200 status code to client
		_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			log.Error(err.Error())
			return
		}
	} else {
		// not https, sent the request header to remote
		_, err = rawHTTPRequestHeader.WriteTo(remoteConn)
		if err != nil {
			log.Error(err.Error())
			return
		}
	}

	// build bidirectional-streams
	log.Info("begin tunnel:", clientConn.RemoteAddr(), "<->", remote)
	proxy.tunnel(clientConn, remoteConn)
	log.Info("stop tunnel:", clientConn.RemoteAddr(), "<->", remote)
}

// extractTunnelInfo extract tunnel info from cient request
func (proxy *dooProxy) extractTunnelInfo(conn net.Conn) (rawReqHeader bytes.Buffer, host, credential string, isHTTPS bool, err error) {
	br := bufio.NewReader(conn)
	tp := textproto.NewReader(br)

	// http proto: GET /index.html HTTP/1.0
	requestLine, err := tp.ReadLine()
	if err != nil {
		return
	}

	method, requestURI, _, ok := parseRequestLine(requestLine)
	if !ok {
		err = fmt.Errorf("malformed HTTP request")
		return
	}

	// https request
	if method == "CONNECT" {
		isHTTPS = true
		requestURI = "http://" + requestURI
	}

	// get remote host
	uriInfo, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return
	}

	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	credential = mimeHeader.Get("Proxy-Authorization")

	if uriInfo.Host == "" {
		host = mimeHeader.Get("Host")
	} else {
		if strings.Index(uriInfo.Host, ":") == -1 {
			host = uriInfo.Host + ":80"
		} else {
			host = uriInfo.Host
		}
	}

	// rebuild http request header
	rawReqHeader.WriteString(requestLine + "\r\n")
	for k, vs := range mimeHeader {
		for _, v := range vs {
			rawReqHeader.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	rawReqHeader.WriteString("\r\n")
	return
}

func (proxy *dooProxy) tunnel(clientConn net.Conn, remoteConn net.Conn) {
	group := &sync.WaitGroup{}
	group.Add(2)
	go func() {
		defer group.Done()
		_, err := io.Copy(remoteConn, clientConn)
		if err != nil {
			log.Error(err.Error())
		}
	}()
	go func() {
		defer group.Done()
		_, err := io.Copy(clientConn, remoteConn)
		if err != nil {
			log.Error(err.Error())
		}
	}()
	group.Wait()
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

// auth provide basic authentication
func (proxy *dooProxy) auth(conn net.Conn, credential string) bool {
	if proxy.isAuth() && proxy.validateCredential(credential) {
		return true
	}
	_, err := conn.Write(
		[]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"*\"\r\n\r\n"))
	if err != nil {
		log.Error(err.Error())
	}
	return false
}

func (proxy *dooProxy) isAuth() bool {
	return proxy.credential != ""
}

func (proxy *dooProxy) validateCredential(credential string) bool {
	c := strings.Split(credential, " ")
	if len(c) == 2 && strings.EqualFold(c[0], "Basic") && c[1] == proxy.credential {
		return true
	}
	return false
}
