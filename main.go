package main

import (
	"scheule/commands"

	"github.com/mix-go/xcli"
	"github.com/mix-go/xutil/xenv"
	_ "scheule/config/dotenv"
	_ "scheule/di"
)

func main() {
	xcli.SetName("app").
		SetVersion("0.0.0-alpha").
		SetDebug(xenv.Getenv("APP_DEBUG").Bool(false))
	xcli.AddCommand(commands.Commands...).Run()
}
