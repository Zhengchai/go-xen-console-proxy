package main

import "testing"

func TestCreateSession(t *testing.T) {

	//var s *ConsoleSession
	var token = "cDiJpVXbkMSG_GiyISA5WIfiy8UzKzRKV73b4UIpnneIbexXtMzwqKUkQ9NPxh6zivm6Eja29EuQCBq-3I6_oQ0IOpQK3amD5xo6BgBZAM0OTow0zd3e9R5AqQyhqoHYTR0bUe-lxap6bTXrEMY01IKmqc7Kkbqo6tUUdU9Y9-X7HBQfJcvZxA5pX-WQ5c8KRdN5cBfekU-os12vJFbk9lV36DqUQioF2bo5xKu4YHJ0AMUjcavQw3uDUbOpE2Ily1mRm5f7h9HnFyFvVy9Ob5EBOpSxz2KD796r77-dxEofr6f4bBtf_LncKAy9GhaGXrZpWp6UZA0b75_PpUYKXnqZCpXx5Q6-i37kayzeXW-FNnQDCzbNydg-32mbDls2fD14s6a11jgVHrBWpgCAV1z0CX8TWILaBYAm2Z3KRgjKOYeoSs6kwdVASzqvH-RU8-hLem7P_d5u8bB4kdR385k2st-YDMTKZ_ON07JO6KQ"
	var key = "kV9Ld-X4rKlTQF4ZJwyn9A"
	var iv = "PCb_WQYrUgbahQeqDEkuUw"
	_, _ = NewConsoleSession(token, key, iv)

}
