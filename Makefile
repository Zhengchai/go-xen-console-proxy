all:
	esc -o static.go static
	go build
