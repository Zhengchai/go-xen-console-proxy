package main

import "testing"

var key = "kV9Ld-X4rKlTQF4ZJwyn9A"
var iv = "PCb_WQYrUgbahQeqDEkuUw"

func TestEncryption(t *testing.T) {
	var plaintext = "test"
	var expected = "FRhtzKa23rouVTFbq3chZg"

	result, _ := encrypt(key, iv, plaintext)
	if result != expected {
		t.Error("Expected:" + expected + " Got:" + result)
	}
}
