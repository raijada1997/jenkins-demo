package main

import (
	"dashboard-demo/elastic"
	_ "dashboard-demo/routers"

	"github.com/beego/beego/v2/server/web"
)

func main() {

	web.BConfig.CopyRequestBody = true

	elastic.InitElastic()
	web.Run()
}
