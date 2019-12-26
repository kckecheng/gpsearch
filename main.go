package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kckecheng/gpsearch/search"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "gpsearch",
		Usage: "search golang packages",
		UsageText: `
			search [--sort|-s <field>] [--reverse|-r] [--num|-n <Num. of packages to list>] \
				[[--fields|-f <field1>] [--fields|-f <field2>] [...]] <pattern1> [pattern2] [...]`,
		Description: `
			Search golang packages on godoc.org from CLI

			Fields supported:
				name        : The name
				path        : The import path
				synopsis    : The description
				import_count: Num. of projects using/importing this
				stars       : Num. of github stars
				score       : How well is the document
				fork        : If this is a forked repository

			Notes: Some packages only have few fields with data
		`,
		Version: "1.2.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "sort",
				Aliases: []string{"s"},
				Value:   "import_count",
				Usage:   "Sort packages based on",
			},
			&cli.BoolFlag{
				Name:    "reverse",
				Aliases: []string{"r"},
				Value:   false,
				Usage:   "Reverse the sort result",
			},
			&cli.UintFlag{
				Name:    "num",
				Aliases: []string{"n"},
				Value:   10,
				Usage:   "Num. of packages to list",
			},
			&cli.StringSliceFlag{
				Name:    "fields",
				Aliases: []string{"f"},
				Value:   cli.NewStringSlice("path", "import_count", "synopsis"),
				Usage:   "Package fields to show, speficie multiple values with -f <field1> -f <field2> ...",
			},
		},
		Action: func(c *cli.Context) error {
			// Exit if no query string is provided
			if c.NArg() == 0 {
				fmt.Printf("\nNo query string is provided\n\n")
				cli.ShowAppHelpAndExit(c, search.ErrExit)
			}

			// Query for packages
			qstr := strings.Join(c.Args().Slice(), "20%")
			s, err := search.NewSearch(qstr)
			if err != nil {
				return err
			}

			// Sort packages
			s.Sort(c.String("sort"), c.Bool("reverse"))

			// Output
			s.FormatOutN(c.StringSlice("fields"), int(c.Uint("num")))
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
