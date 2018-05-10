package main

import (
	"log"

	"landzero.net/x/net/web"
	"landzero.net/x/net/web/binding"
)

// Item item of debug message
type Item struct {
	UniqueID string `form:"uniqueId"`
	Message  string `form:"message"`
}

func main() {
	w := web.New()
	w.Use(web.Recovery())
	w.Post("/inlet", binding.Form(Item{}), func(ctx *web.Context, it Item) {
		log.Println(it.UniqueID, ":", it.Message)
	})
	w.Run()
}
