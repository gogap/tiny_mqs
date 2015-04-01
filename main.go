package main

import (
	"github.com/gogap/tiny_mqs/mqs"
)

func main() {
	mqs := mqs.NewTinyMQS()
	mqs.Run()
}
