package main

import (
	"github.com/sunbk201/ua3f/cmd"
)

var appVersion = "Development"

func main() {
	cmd.AppVersion = appVersion
	cmd.Execute()
}
