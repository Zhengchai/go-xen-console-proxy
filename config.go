package main

import (
	"gopkg.in/gcfg.v1"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Server configServer
}

type configServer struct {
	Port          int
	Hostname      string
	EncryptionKey string
	EncryptionIv  string
}

func (c *configServer) Addr() string {
	return c.Hostname + ":" + strconv.Itoa(c.Port)
}

func (c *configServer) SetEncryptionKey(key string) {
	c.EncryptionKey = key
}

func (c *configServer) SetEncryptionIv(iv string) {
	c.EncryptionIv = iv
}

const defaultConfig = `
	[server]
	port=9090
	hostname=0.0.0.0
`

func init() {
	logger = log.New(os.Stdout, "[websockify] ", log.Ldate|log.Ltime)

	err = gcfg.ReadStringInto(&cfg, defaultConfig)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
