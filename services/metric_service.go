package services

import (
	"dashboard-demo/elastic"
	"dashboard-demo/models"
	"log"
	"strings"
	"time"
)

// --------------------------------------------------
// Determine Operational Category
// --------------------------------------------------

func determineOperationalCategory(pipeline string) string {

	pipeline = strings.ToLower(pipeline)

	if strings.Contains(pipeline, "fusion-hci-upgrade") {
		return "UPGRADE"
	}

	if strings.Contains(pipeline, "hci-rack-automation") {
		return "INSTALL"
	}

	return "OTHER"
}

func normalizeStageName(stage string) string {

	stage = strings.ToLower(stage)
	stage = strings.ReplaceAll(stage, " ", "_")

	return stage
}

// --------------------------------------------------
// Process Incoming Payload
// --------------------------------------------------

func ProcessFusionMetric(payload models.FusionPayload) error {

	// ----------------------------------
	// Ensure timestamp
	// ----------------------------------
	if payload.Metadata.Timestamp == "" {
		payload.Metadata.Timestamp =
			time.Now().UTC().Format(time.RFC3339)
	}

	// ----------------------------------
	// Determine operational category
	// ----------------------------------
	payload.OperationalCategory =
		determineOperationalCategory(payload.Job.PipelineName)

	buildID := payload.Job.JenkinsBuildID

	// =========================================
	// STEP 1: HANDLE RETRY FIRST
	// =========================================
	if payload.Job.Retry {

		log.Println("Retry detected. Finding latest build to update...")

		docID, existingDoc, err :=
			elastic.FindLatestByRack(
				payload.Job.RackName,
				payload.Job.PipelineName,
				payload.OperationalCategory,
			)

		if err != nil {
			log.Println("Retry failed: no previous document found")
			return err
		}

		merged := mergeBuild(existingDoc, payload)

		// Append retry build id
		prevJobs, ok := merged["previous_jobs"].([]interface{})
		if !ok {
			prevJobs = []interface{}{}
		}

		prevJobs = append(prevJobs, buildID)
		merged["previous_jobs"] = prevJobs

		return elastic.InsertDocumentWithID(docID, merged)
	}

	// =========================================
	// STEP 2: CHECK IF SAME BUILD EXISTS
	// =========================================
	existingDoc, err := elastic.GetDocumentByID(buildID)

	if err == nil {

		log.Println("Existing build found, merging build_id:", buildID)

		merged := mergeBuild(existingDoc, payload)

		return elastic.InsertDocumentWithID(buildID, merged)
	}

	// =========================================
	// STEP 3: CREATE NEW BUILD
	// =========================================

	log.Println("Creating new document for build_id:", buildID)

	stageMap := make(map[string]interface{})

	// INSTALL stage order (for manual injection)
	installStages := []string{
		"pull_iso",
		"mgen_cleanup",
		"stage_1",
		"stage_2",
		"storage",
		"services",
		"fvt",
		"bvt",
		"svt",
	}

	// ----------------------------------
	// Inject manual previous stages
	// Only for fresh INSTALL builds
	// ----------------------------------
	if payload.OperationalCategory == "INSTALL" &&
		!payload.Job.Retry {

		startStage := normalizeStageName(payload.Job.StartStage)

		for _, stage := range installStages {

			if stage == startStage {
				break
			}

			stageMap[stage] = map[string]interface{}{
				"stage_status":     "SUCCESS",
				"duration_seconds": 0,
			}
		}
	}

	// ----------------------------------
	// Add incoming stages from payload
	// ----------------------------------
	for stageName, stage := range payload.Metrics.Stages {

		normalized := normalizeStageName(stageName)

		stageMap[normalized] = map[string]interface{}{
			"stage_status":     stage.StageStatus,
			"duration_seconds": stage.DurationSeconds,
		}
	}

	// ----------------------------------
	// Determine job_status
	// ----------------------------------
	var jobStatus string

	if strings.ToLower(payload.Metrics.PayloadType) == "job" &&
		payload.Metrics.JobStatus != "" {

		jobStatus = normalizeStatus(payload.Metrics.JobStatus)

	} else {

		jobStatus = calculateJobStatus(
			stageMap,
			payload.Job.StopStage,
		)
	}

	// ----------------------------------
	// Build document
	// ----------------------------------
	doc := map[string]interface{}{

		// Job fields
		"pipeline_name":            payload.Job.PipelineName,
		"rack_name":                payload.Job.RackName,
		"rack_ip":                  payload.Job.RackIP,
		"build_id":                 buildID,
		"jenkins_build_url":        payload.Job.JenkinsBuildURL,
		"isf_operator_build_id":    payload.Job.ISFOperatorBuildID,
		"fusion_version":           payload.Job.FusionVersion,
		"ocp_version":              payload.Job.OCPVersion,
		"environment":              payload.Job.Environment,
		"lab_location":             payload.Job.LabLocation,
		"install_type":             payload.Job.InstallType,
		"storage_type":             payload.Job.StorageType,
		"topology":                 payload.Job.Topology,
		"rack_type":                payload.Job.RackType,
		"is_production":            payload.Job.IsProduction,
		"production_version":       payload.Job.ProductionVersion,
		"node_configuration":       payload.Job.NodeConfiguration,
		"download_iso":             payload.Job.DownloadISO,
		"iso_build":                payload.Job.ISOBuild,
		"upgrade_ocp":              payload.Job.UpgradeOCP,
		"offline_install":          payload.Job.OfflineInstall,
		"is_reinstall":             payload.Job.IsReinstall,
		"skip_bvt":                 payload.Job.SkipBVT,
		"skip_fvt":                 payload.Job.SkipFVT,
		"bvt_build":                payload.Job.BVTBuild,
		"svt_suite":                payload.Job.SVTSuite,
		"df_test_suite":            payload.Job.DFTestSuite,
		"services_to_install":      payload.Job.ServicesToInstall,
		"enable_service_framework": payload.Job.EnableServiceFramework,
		"run_util_calc":            payload.Job.RunUtilCalc,
		"enable_mgen_update":       payload.Job.EnableMgenUpdate,
		"automation_branch":        payload.Job.AutomationBranch,
		"fusion_deploy_branch":     payload.Job.FusionDeployBranch,
		"devops_branch":            payload.Job.DevopsBranch,
		"triggered_by":             payload.Job.TriggeredBy,

		// System fields
		"retry":                false,
		"previous_jobs":        []string{},
		"stages":               stageMap,
		"job_status":           jobStatus,
		"operational_category": payload.OperationalCategory,
		"timestamp":            payload.Metadata.Timestamp,
		"source":               payload.Metadata.Source,
		"version":              payload.Metadata.Version,
	}

	return elastic.InsertDocumentWithID(buildID, doc)
}
