package commands

import (
	"github.com/mix-go/xcli"
)

var Commands = []*xcli.Command{
	{
		Name:  "hello",
		Short: "\tEcho demo",
		Options: []*xcli.Option{
			{
				Names: []string{"n", "name"},
				Usage: "Your name",
			},
			{
				Names: []string{"say"},
				Usage: "\tSay ...",
			},
		},
		RunI: &HelloCommand{},
	},
	{
		Name:  "scheduler",
		Short: "\tRun cron scheduler",
		Options: []*xcli.Option{
			{
				Names: []string{"base"},
				Usage: "Override task API base url",
			},
		},
		RunI: &SchedulerCommand{},
	},
	{
		Name:  "web",
		Short: "\tRun Layui admin UI",
		Options: []*xcli.Option{
			{
				Names: []string{"addr"},
				Usage: "Listen address, default :8080",
			},
		},
		RunI: &WebCommand{},
	},
}
