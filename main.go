package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
)

var (
	cfg    Config
	logger *log.Logger
	err    error
	host   string
	port   string
)

type EncryptorSecret struct {
	Key string `json:"base64EncodedKeyBytes"`
	Iv  string `json:"base64EncodedIvBytes"`
}

// Given a local session, establish a Websocket <-> HTTPS tunnel to the
// Xenserver
func handleVncWebsocketProxy(w http.ResponseWriter, r *http.Request) {

	// XXX: With the current implementation, anyone who has a valid sessionID
	// can gain access to the VNC

	log.Printf("VNC request " + r.URL.String())
	paths := strings.Split(r.URL.Path, "/")

	if len(paths) < 3 || SessionMap[paths[2]] == nil {
		mesg := "Unable to find session"
		log.Printf(mesg)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := paths[2]

	session := SessionMap[sessionID]
	log.Printf("session:%+v\n", session)

	//if there is a previous session running, close it
	if session.wsConn != nil || session.tlsConn != nil {
		session.wsConn.Close()
		session.tlsConn.Close()
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	h := http.Header{}
	h.Set("Sec-WebSocket-Protocol", "binary")

	wsConn, err := upgrader.Upgrade(w, r, h)
	if err != nil {
		delete(SessionMap, sessionID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	xenConn, err := initXenConnection(session)
	if err != nil {
		delete(SessionMap, sessionID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.wsConn = wsConn
	session.tlsConn = xenConn

	proxy := NewProxyServer(sessionID, wsConn, xenConn)
	proxy.DoProxy()
}

// Decypt and get the tunnel URL and xenserver session ID, setup a new local session
// with all the variables and serve the vnc.html.
func handleNewConsoleConnection(w http.ResponseWriter, r *http.Request) {
	logger.Printf("new connection from: %s", r.RemoteAddr)
	path := r.URL.Query().Get("path")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if path == "" {

		token := r.URL.Query().Get("token")
		consoleSession, err := NewConsoleSession(cfg.Server.EncryptionKey, cfg.Server.EncryptionIv, token)
		if err != nil {
			log.Printf("error creating console session " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		isValid := consoleSession.Validate()
		if !isValid {
			log.Printf(" invalid session ")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sessionId := consoleSession.GenerateUuid()
		log.Printf("Setting new session:" + sessionId)
		SessionMap[sessionId] = consoleSession
		http.Redirect(w, r, "/static/vnc.html?path="+sessionId, http.StatusFound)

	} else {

		log.Printf("got session %s ", path)
		consoleSession := SessionMap[path]

		if consoleSession == nil {
			log.Printf("Unable to find session for " + path)
			http.Error(w, "Not Found", http.StatusInternalServerError)
		}

		http.ServeFile(w, r, "static/vnc.html")
	}

}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	log.Printf("static get : %s", r.URL.Path[1:])
	http.ServeFile(w, r, r.URL.Path[1:])
}

//We get the encryption key from the console proxy
func handleSetEncryptorPassword(w http.ResponseWriter, r *http.Request) {

	//Only allow this from localhost
	parts := strings.Split(r.RemoteAddr, ":")
	if parts[0] != "127.0.0.1" {
		log.Printf("Request from non-local host")
		return
	}

	if secretJson := r.URL.Query().Get("secret"); secretJson != "" {
		var secret EncryptorSecret
		log.Println("Secret json: " + secretJson)
		err := json.Unmarshal([]byte(secretJson), &secret)
		if err != nil {
			log.Printf("Unable to decode the secret sent from the servelet")
			return
		}
		cfg.Server.SetEncryptionKey(secret.Key)
		cfg.Server.SetEncryptionIv(secret.Iv)
		fmt.Fprintf(w, "Password was set %s:%s ", secret.Key, secret.Iv)
	}

	log.Printf("Password was set\n")
}

func initXenConnection(session *ConsoleSession) (*tls.Conn, error) {

	if session.ClientTunnelSession == "" || session.ClientTunnelUrl == "" {
		mesg := "Unable to find Tunnel URL or Tunnel Session"
		log.Printf(mesg)
		return nil, errors.New(mesg)
	}

	//open session to Xenserver
	tunnelUrl, err := url.Parse(session.ClientTunnelUrl)
	if err != nil {
		mesg := "Unable to parse session URL"
		log.Printf(mesg)
		return nil, errors.New(mesg)
	}

	host := tunnelUrl.Host
	uri := tunnelUrl.RequestURI()

	data := fmt.Sprintf("CONNECT %s HTTP/1.0\r\nHost: %s\r\nCookie: session_id=%s\r\n\r\n",
		uri, host, session.ClientTunnelSession)

	xenConn, err := tls.Dial("tcp", host+":443", &tls.Config{InsecureSkipVerify: true})
	_, err = xenConn.Write([]byte(data))
	if err != nil {
		logger.Println(err.Error())
		return nil, err
	}

	reader := bufio.NewReader(xenConn)

	success := false
	for {
		l, _, err := reader.ReadLine()
		line := string(l)
		if err != nil {
			log.Printf("Error reading data from xenserver " + err.Error())
			return nil, err
		}
		fmt.Println(line)
		if line == "HTTP/1.1 200 OK" {
			success = true
		}

		if line == "" {
			break
		}
	}

	if !success {
		mesg := "non 200 response from xenserver https"
		log.Printf(mesg)
		return nil, errors.New(mesg)
	}

	return xenConn, nil
}

func main() {

	logger.Printf("listening on %s\n", cfg.Server.Addr())
	http.HandleFunc("/console", handleNewConsoleConnection)
	http.HandleFunc("/setEncryptorPassword", handleSetEncryptorPassword)
	http.Handle("/static/", http.FileServer(FS(false)))
	http.HandleFunc("/vnc/", handleVncWebsocketProxy)
	http.ListenAndServe(cfg.Server.Addr(), context.ClearHandler(http.DefaultServeMux))
}
