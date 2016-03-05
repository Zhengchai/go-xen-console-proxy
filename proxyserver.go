package main

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
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
		fmt.Printf("x")
		if err != nil {
			log.Printf(err.Error())
			proxyserver.tlsConn.Close()
			proxyserver.wsConn.Close()
			break
		}

		err = proxyserver.wsConn.WriteMessage(websocket.BinaryMessage, buffer[0:n])
		fmt.Printf("W")
		if err != nil {
			log.Println(err.Error())
			proxyserver.tlsConn.Close()
			proxyserver.wsConn.Close()
			break
		}
	}
}

func (proxyserver *ProxyServer) wsToTcp() {
	for {
		_, data, err := proxyserver.wsConn.ReadMessage()
		fmt.Printf("w")
		fmt.Printf(string(data))
		if err != nil {
			log.Println(err.Error())
			proxyserver.wsConn.Close()
			proxyserver.tlsConn.Close()
			break
		}

		_, err = proxyserver.tlsConn.Write(data)
		fmt.Printf("X")
		if err != nil {
			log.Println(err.Error())
			proxyserver.wsConn.Close()
			proxyserver.tlsConn.Close()
			break
		}
	}
}
