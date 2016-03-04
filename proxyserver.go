package main

import (
	"crypto/tls"
	"github.com/gorilla/websocket"
)

type ProxyServer struct {
	wsConn  *websocket.Conn
	tlsConn *tls.Conn
}

func NewProxyServer(wsConn *websocket.Conn, tlsConn *tls.Conn) *ProxyServer {
	proxyserver := ProxyServer{wsConn, tlsConn}
	return &proxyserver
}

func (proxyserver *ProxyServer) DoProxy() {
	go proxyserver.wsToTcp()
	proxyserver.tcpToWs()
}

func (proxyserver *ProxyServer) tcpToWs() {
	buffer := make([]byte, 1024)

	for {
		n, err := proxyserver.tlsConn.Read(buffer)
		if err != nil {
			proxyserver.tlsConn.Close()
			break
		}

		err = proxyserver.wsConn.WriteMessage(websocket.BinaryMessage, buffer[0:n])
		if err != nil {
			logger.Println(err.Error())
		}
	}
}

func (proxyserver *ProxyServer) wsToTcp() {
	for {
		_, data, err := proxyserver.wsConn.ReadMessage()
		if err != nil {
			break
		}

		_, err = proxyserver.tlsConn.Write(data)
		if err != nil {
			logger.Println(err.Error())
			break
		}
	}
}
