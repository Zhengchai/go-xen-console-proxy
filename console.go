package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

var (
	cfg         Config
	err         error
	logger      *log.Logger
	host        string
	port        string
	proxyserver *ProxyServer
)

var sessionstore = sessions.NewCookieStore([]byte("something-very-secret"))

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Given a local session, establish a Websocket <-> HTTPS tunnel to the
// Xenserver
func handleClientWebsocketProxy(w http.ResponseWriter, r *http.Request) {

	wsConn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logger.Printf(err.Error())
		return
	}

	dataType, data, err := wsConn.ReadMessage()
	if err != nil {
		_ = wsConn.WriteMessage(dataType, []byte("Fail: "+err.Error()))
		return
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", string(data))
	if err != nil {
		errorMsg := "FAIL(net resolve tcp addr): " + err.Error()
		logger.Println(errorMsg)
		_ = wsConn.WriteMessage(websocket.CloseMessage, []byte(errorMsg))
		return
	}

	tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		errorMsg := "FAIL(net dial tcp): " + err.Error()
		logger.Println(errorMsg)
		_ = wsConn.WriteMessage(websocket.CloseMessage, []byte(errorMsg))
		return
	}

	proxyserver = NewProxyServer(wsConn, tcpConn)
	go proxyserver.doProxy()

}

// Decypt and get the tunnel URL and xenserver session ID, setup a new local session
// with all the variables and serve the vnc.html.
func handleNewConnection(w http.ResponseWriter, r *http.Request) {
	logger.Printf("new connection from: %s", r.RemoteAddr)
	session_id := r.URL.Query().Get("id")

	session, err := sessionstore.Get(r, session_id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if session.Values["id"] == nil {
		log.Printf("Setting session !!!")
		//decrypt and redirect

		session.Values["id"] = session_id
		session.Save(r, w)

		http.Redirect(w, r, "/vnc.html", http.StatusFound)

	} else {
		log.Printf("got session %s ", session.Values["id"])
	}

}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

func handleSetEncryptionKey(w http.ResponseWriter, r *http.Request) {
	logger.Printf("setting encryption key from %s", r.RemoteAddr)
	parts := strings.Split(r.RemoteAddr, ":")

	//Only allow this from localhost
	if parts[0] != "127.0.0.1" {
		return
	}

	queryVals := r.URL.Query()

	if key := queryVals.Get("key"); key != "" {
		cfg.Server.SetEncryptionKey(key)
	}
}

func main() {

	logger.Printf("listening on %s\n", cfg.Server.Addr())
	http.HandleFunc("/novnc", handleNewConnection)
	http.HandleFunc("/setEncryptionKey", handleSetEncryptionKey)
	http.HandleFunc("/include", handleStatic)
	http.ListenAndServe(cfg.Server.Addr(), context.ClearHandler(http.DefaultServeMux))
}
