package main

import (
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "analyst"
	app.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "creates a spreadsheet out of a script",
			Action:  Create,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "script",
					Value: "",
					Usage: "path to script",
				},
			},
		},
		{
			Name:    "validate",
			Aliases: []string{"v"},
			Usage:   "validates a script",
			Action:  Validate,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "script",
					Value: "",
					Usage: "path to script",
				},
			},
		},
	}
	app.Run(os.Args)

}