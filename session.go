package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/url"
	"regexp"

	"github.com/gorilla/websocket"
)

type ConsoleSession struct {
	ClientHostAddress   string `json:"clientHostAddress"`
	ClientHostPort      int    `json:"clientHostPort"`
	ClientHostPassword  string `json:"clientHostPassword"`
	ClientTag           string `json:"clientTag"`
	Ticket              string `json:"ticket"`
	locale              string `json:"locale"`
	ClientTunnelUrl     string `json:"clientTunnelUrl"`
	ClientTunnelSession string `json:"clientTunnelSession"`

	wsConn  *websocket.Conn
	tlsConn *tls.Conn
}

var SessionMap map[string]*ConsoleSession = make(map[string]*ConsoleSession)

// Decrypts a token string and returns a session struct
func NewConsoleSession(key, iv, token string) (*ConsoleSession, error) {
	decrypted, err := decrypt(key, iv, token)
	if err != nil {
		log.Println("Error decrypting")
		return nil, err
	}
	var session ConsoleSession
	err = json.Unmarshal([]byte(decrypted), &session)
	if err != nil {
		log.Println("Error decoding json")
		return nil, err
	}

	return &session, nil
}

func (s *ConsoleSession) Validate() bool {
	//check if a valid session is given
	r := regexp.MustCompile("^OpaqueRef:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")
	if !r.MatchString(s.ClientTunnelSession) {
		return false
	}

	//check if the URL is valid
	tunnelUrl, err := url.Parse(s.ClientTunnelUrl)
	if err != nil {
		return false
	}

	//check if query contains console uuid
	consoleUUID := tunnelUrl.Query().Get("uuid")
	if consoleUUID == "" {
		return false
	}

	//check if the console UUID is a valid UUID
	r = regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")
	if !r.MatchString(consoleUUID) {
		return false
	}

	return true
}

//The UUID for a session is the SHA256 of its TunnelSession and TunnelURL
func (s *ConsoleSession) GenerateUuid() string {
	hash := sha256.Sum256([]byte(s.ClientTunnelUrl + s.ClientTunnelSession))
	return base64.URLEncoding.EncodeToString(hash[:])

}
