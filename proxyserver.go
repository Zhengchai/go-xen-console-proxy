package main

import (
	"crypto/tls"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

type ProxyServer struct {
	sessionID string
	wsConn    *websocket.Conn
	tlsConn   *tls.Conn
}

func NewProxyServer(sessionID string, wsConn *websocket.Conn, tlsConn *tls.Conn) *ProxyServer {
	proxyserver := ProxyServer{sessionID, wsConn, tlsConn}
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
			log.Printf(err.Error())

			log.WithFields(logrus.Fields{
				"err":         err,
				"proxyserver": proxyserver,
			}).Warn("Error reading from TLS")

			delete(SessionMap, proxyserver.sessionID)
			proxyserver.tlsConn.Close()
			proxyserver.wsConn.Close()
			break
		}

		err = proxyserver.wsConn.WriteMessage(websocket.BinaryMessage, buffer[0:n])
		if err != nil {

			log.WithFields(logrus.Fields{
				"err":         err,
				"proxyserver": proxyserver,
			}).Warn("Error writing to websocket")

			delete(SessionMap, proxyserver.sessionID)
			proxyserver.tlsConn.Close()
			proxyserver.wsConn.Close()
			break
		}
	}
}

func (proxyserver *ProxyServer) wsToTcp() {
	for {
		_, data, err := proxyserver.wsConn.ReadMessage()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":         err,
				"proxyserver": proxyserver,
			}).Warn("Error reading from websocket")

			delete(SessionMap, proxyserver.sessionID)
			proxyserver.wsConn.Close()
			proxyserver.tlsConn.Close()
			break
		}

		_, err = proxyserver.tlsConn.Write(data)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":         err,
				"proxyserver": proxyserver,
			}).Warn("Error writing to tls")

			delete(SessionMap, proxyserver.sessionID)
			proxyserver.wsConn.Close()
			proxyserver.tlsConn.Close()
			break
		}
	}
}
