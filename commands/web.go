package commands

import (
	"scheule/di"
	"scheule/web"

	"github.com/mix-go/xcli/flag"
)

type WebCommand struct{}

func (t *WebCommand) Main() {
	addr := flag.Match("addr").String(":8080")
	logger := di.Zap()
	db := di.Gorm()

	server := web.NewAdminServer(db, logger)
	if err := server.Start(addr); err != nil {
		logger.Fatalf("admin server exited: %v", err)
	}
}
