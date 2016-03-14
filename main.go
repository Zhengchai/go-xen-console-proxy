package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
)

var (
	cfg  Config
	err  error
	host string
	port string
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

	log.WithFields(logrus.Fields{
		"url": r.URL.String(),
	}).Debug("New VNC session")

	paths := strings.Split(r.URL.Path, "/")

	if len(paths) < 3 || SessionMap[paths[2]] == nil {
		mesg := "Unable to find session"

		log.WithFields(logrus.Fields{
			"url": r.URL.String(),
		}).Warn(mesg)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := paths[2]
	session := SessionMap[sessionID]

	log.WithFields(logrus.Fields{
		"session": session,
	}).Debug("Found session")

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

		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Error upgrading wesocket")

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	xenConn, err := initXenConnection(session)
	if err != nil {
		delete(SessionMap, sessionID)

		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Error initalizing xenserver tunnel")

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

	log.WithFields(logrus.Fields{
		"remotehost": r.RemoteAddr,
	}).Debug("New connection")

	path := r.URL.Query().Get("path")

	if path == "" {

		token := r.URL.Query().Get("token")
		consoleSession, err := NewConsoleSession(cfg.Server.EncryptionKey, cfg.Server.EncryptionIv, token)

		if err != nil {
			mesg := "error creating console session "
			log.WithFields(logrus.Fields{
				"error": err,
			}).Warn(mesg)

			http.Error(w, mesg, http.StatusInternalServerError)
			return
		}

		isValid := consoleSession.Validate()
		if !isValid {

			log.WithFields(logrus.Fields{
				"session": consoleSession,
			}).Warn("Error validating session")

			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sessionId := consoleSession.GenerateUuid()

		log.WithFields(logrus.Fields{
			"session_id": sessionId,
		}).Debug("Starting a new session")

		SessionMap[sessionId] = consoleSession
		http.Redirect(w, r, "/static/vnc.html?path="+sessionId, http.StatusFound)

	} else {

		log.WithFields(logrus.Fields{
			"path": path,
		}).Debug("Got a new session")

		consoleSession := SessionMap[path]

		if consoleSession == nil {

			log.WithFields(logrus.Fields{
				"path": path,
			}).Debug("Unable to find session")

			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

		http.ServeFile(w, r, "static/vnc.html")
	}

}

func handleStatic(w http.ResponseWriter, r *http.Request) {

	log.WithFields(logrus.Fields{
		"path": r.URL.Path[1:],
	}).Debug("Static file request")

	http.ServeFile(w, r, r.URL.Path[1:])
}

//We get the encryption key from the console proxy
func handleSetEncryptorPassword(w http.ResponseWriter, r *http.Request) {

	//Only allow this from localhost
	parts := strings.Split(r.RemoteAddr, ":")
	if parts[0] != "127.0.0.1" {

		log.WithFields(logrus.Fields{
			"host": parts[0],
		}).Warn("Non-local request to set password")

		return
	}

	if secretJson := r.URL.Query().Get("secret"); secretJson != "" {
		var secret EncryptorSecret
		err := json.Unmarshal([]byte(secretJson), &secret)
		if err != nil {

			log.WithFields(logrus.Fields{
				"secret_json": secretJson,
			}).Warn("Unable to decode the secret sent from the servelet")

			return
		}
		cfg.Server.SetEncryptionKey(secret.Key)
		cfg.Server.SetEncryptionIv(secret.Iv)
		fmt.Fprintf(w, "Password was set %s:%s ", secret.Key, secret.Iv)
	}

	log.Debug("The password was set")
}

func initXenConnection(session *ConsoleSession) (*tls.Conn, error) {

	if session.ClientTunnelSession == "" || session.ClientTunnelUrl == "" {
		mesg := "Unable to find Tunnel URL or Tunnel Session"

		log.WithFields(logrus.Fields{
			"session": session,
		}).Warn(mesg)

		return nil, errors.New(mesg)
	}

	//open session to Xenserver
	tunnelUrl, err := url.Parse(session.ClientTunnelUrl)
	if err != nil {

		mesg := "Unable to parse session URL"
		log.WithFields(logrus.Fields{
			"tunnel_url": session.ClientTunnelUrl,
			"error":      err,
		}).Warn(mesg)

		return nil, errors.New(mesg)
	}

	host := tunnelUrl.Host
	uri := tunnelUrl.RequestURI()

	data := fmt.Sprintf("CONNECT %s HTTP/1.0\r\nHost: %s\r\nCookie: session_id=%s\r\n\r\n",
		uri, host, session.ClientTunnelSession)

	xenConn, err := tls.Dial("tcp", host+":443", &tls.Config{InsecureSkipVerify: true})
	_, err = xenConn.Write([]byte(data))
	if err != nil {

		log.WithFields(logrus.Fields{
			"error":   err,
			"session": session,
		}).Warn("Failed to connect to Xenserver")

		return nil, err
	}

	reader := bufio.NewReader(xenConn)

	success := false
	for {
		l, _, err := reader.ReadLine()
		line := string(l)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":   err,
				"session": session,
			}).Warn("Error reading data from xenserver")

			return nil, err
		}
		log.Debug(line)
		if line == "HTTP/1.1 200 OK" {
			success = true
		}

		if line == "" {
			break
		}
	}

	if !success {
		mesg := "non 200 response from xenserver https"

		log.Warn(mesg)
		return nil, errors.New(mesg)
	}

	return xenConn, nil
}

func main() {

	log.WithFields(logrus.Fields{
		"addr": cfg.Server.Addr(),
	}).Info("Listening")

	http.HandleFunc("/console", handleNewConsoleConnection)
	http.HandleFunc("/setEncryptorPassword", handleSetEncryptorPassword)
	http.Handle("/static/", http.FileServer(FS(false)))
	http.HandleFunc("/vnc/", handleVncWebsocketProxy)
	http.ListenAndServe(cfg.Server.Addr(), context.ClearHandler(http.DefaultServeMux))
}
