package controllers

import (
	"dashboard-demo/models"
	"dashboard-demo/services"
	"encoding/json"
	"log"

	"github.com/beego/beego/v2/server/web"
)

type MetricController struct {
	web.Controller
}
type HealthController struct {
	web.Controller
}

func (c *HealthController) Get() {
	c.Ctx.WriteString("OK")
}

func (c *MetricController) RecordMetric() {

	var payload models.FusionPayload

	log.Println("RAW BODY:", string(c.Ctx.Input.RequestBody))

	err := json.Unmarshal(c.Ctx.Input.RequestBody, &payload)

	if err != nil {
		log.Println("JSON Parse error:", err)
		c.Ctx.Output.SetStatus(400)
		c.Ctx.Output.Body([]byte("Invalid JSON"))
		return
	}

	log.Println("Pipeline:", payload.Job.PipelineName)
	log.Println("Rack:", payload.Job.RackName)
	log.Println("Build ID:", payload.Job.JenkinsBuildID)
	log.Println("Retry:", payload.Job.Retry)
	log.Println("Payload Type:", payload.Metrics.PayloadType)
	log.Println("Incoming Stage Status:", payload.Metrics.StageStatus)
	log.Println("Incoming Job Status:", payload.Metrics.JobStatus)

	// FIXED LOOP
	for stageName, stage := range payload.Metrics.Stages {

		log.Println("Stage:", stageName,
			"Status:", stage.StageStatus,
			"Duration:", stage.DurationSeconds)
	}

	err = services.ProcessFusionMetric(payload)

	if err != nil {

		log.Println("ERROR FROM SERVICE:", err)

		c.Ctx.Output.SetStatus(500)
		c.Ctx.Output.Body([]byte("Failed to store metric"))

		return
	}

	c.Ctx.Output.SetStatus(200)
	c.Ctx.Output.Body([]byte("Metric recorded successfully"))
}
