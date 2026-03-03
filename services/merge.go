package services

import (
	"dashboard-demo/models"
	"strings"
)

func mergeBuild(existing map[string]interface{}, payload models.FusionPayload) map[string]interface{} {

	// Retry handling
	if payload.Job.Retry {
		prevJobs, ok := existing["previous_jobs"].([]interface{})
		if !ok {
			prevJobs = []interface{}{}
		}

		incomingBuildID := payload.Job.JenkinsBuildID

		alreadyExists := false
		for _, v := range prevJobs {
			if v.(string) == incomingBuildID {
				alreadyExists = true
				break
			}
		}

		if !alreadyExists {
			prevJobs = append(prevJobs, incomingBuildID)
		}

		existing["previous_jobs"] = prevJobs
	}

	// Update timestamp
	existing["timestamp"] = payload.Metadata.Timestamp

	// Merge stages
	existingStages, ok := existing["stages"].(map[string]interface{})
	if !ok {
		existingStages = make(map[string]interface{})
	}

	for stageName, stage := range payload.Metrics.Stages {
		existingStages[stageName] = map[string]interface{}{
			"stage_status":     stage.StageStatus,
			"duration_seconds": stage.DurationSeconds,
		}
	}

	existing["stages"] = existingStages

	// Recalculate job_status
	existing["job_status"] = calculateJobStatus(
		existingStages,
		payload.Job.StopStage,
	)

	return existing
}

// ----------------------------------
// Calculate job_status dynamically
// ----------------------------------

func calculateJobStatus(
	stages map[string]interface{},
	stopStage string,
) string {

	hasFailure := false
	stopStageSuccess := false

	for stageName, v := range stages {

		stageData, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		status, ok := stageData["stage_status"].(string)
		if !ok {
			continue
		}

		status = normalizeStatus(status)

		if status == "FAILED" {
			hasFailure = true
		}

		if stageName == stopStage && status == "SUCCESS" {
			stopStageSuccess = true
		}
	}

	if hasFailure {
		return "FAILED"
	}

	if stopStageSuccess {
		return "SUCCESS"
	}

	return "IN-PROGRESS"
}

// ----------------------------------
// Normalize status values
// ----------------------------------

func normalizeStatus(status string) string {

	status = strings.ToUpper(status)

	switch status {
	case "SUCCESS":
		return "SUCCESS"
	case "FAILURE", "FAILED":
		return "FAILED"
	case "IN-PROGRESS", "IN_PROGRESS":
		return "IN-PROGRESS"
	default:
		return status
	}
}
