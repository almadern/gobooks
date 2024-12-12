package main

import (
	"gobook/arguments"
	db "gobook/database"
	"gobook/telegram"
	"gobook/web"
	"log"
)

func init() {
	arguments.Initialization()
	err := db.InitDB()
	if err != nil {
		log.Fatalf("Can't start Initialization: %v", err)
	}
}

func main() {
	if arguments.Config.Telegram && arguments.Config.Web {
		go telegram.BotStart()
		web.StartWeb()
	} else if arguments.Config.Telegram {
		telegram.BotStart()
	} else if arguments.Config.Web {
		web.StartWeb()
	} else {
		log.Panicf("Nothing is chosen.")
	}
}
