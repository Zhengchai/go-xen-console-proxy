package main

import (
	"encoding/json"
	"log"
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
