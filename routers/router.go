package routers

import (
	"dashboard-demo/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/record-metric", &controllers.MetricController{}, "post:RecordMetric")
	web.Router("/health", &controllers.HealthController{})
}
