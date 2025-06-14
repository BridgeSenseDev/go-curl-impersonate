package main

import (
	curl "github.com/BridgeSenseDev/go-curl-impersonate"
)

func main() {
	easy := curl.EasyInit()
	defer easy.Cleanup()
	if easy != nil {
		easy.Setopt(curl.OPT_URL, "http://www.google.com/")
		easy.Perform()
	}
}
