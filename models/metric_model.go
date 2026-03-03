package models

// ------------------
// Job Info
// ------------------

type JobInfo struct {
	PipelineName           string   `json:"pipeline_name"`
	RackName               string   `json:"rack_name"`
	RackIP                 string   `json:"rack_ip"`
	StartStage             string   `json:"start_stage"`
	StopStage              string   `json:"stop_stage"`
	FusionVersion          string   `json:"fusion_version"`
	OCPVersion             string   `json:"ocp_version"`
	Environment            string   `json:"environment"`
	LabLocation            string   `json:"lab_location"`
	InstallType            string   `json:"install_type"`
	StorageType            string   `json:"storage_type"`
	Retry                  bool     `json:"retry"`
	JenkinsBuildID         string   `json:"jenkins_build_id"`
	JenkinsBuildURL        string   `json:"jenkins_build_url"`
	ISFOperatorBuildID     string   `json:"isf_operator_build_id"`
	Topology               string   `json:"topology"`
	RackType               string   `json:"rack_type"`
	IsProduction           bool     `json:"is_production"`
	ProductionVersion      string   `json:"production_version"`
	NodeConfiguration      string   `json:"node_configuration"`
	DownloadISO            string   `json:"download_iso"`
	ISOBuild               string   `json:"iso_build"`
	UpgradeOCP             string   `json:"upgrade_ocp"`
	OfflineInstall         bool     `json:"offline_install"`
	IsReinstall            bool     `json:"is_reinstall"`
	SkipBVT                bool     `json:"skip_bvt"`
	SkipFVT                bool     `json:"skip_fvt"`
	BVTBuild               bool     `json:"bvt_build"`
	SVTSuite               string   `json:"svt_suite"`
	DFTestSuite            string   `json:"df_test_suite"`
	ServicesToInstall      []string `json:"services_to_install"`
	EnableServiceFramework bool     `json:"enable_service_framework"`
	RunUtilCalc            bool     `json:"run_util_calc"`
	EnableMgenUpdate       bool     `json:"enable_mgen_update"`
	AutomationBranch       string   `json:"automation_branch"`
	FusionDeployBranch     string   `json:"fusion_deploy_branch"`
	DevopsBranch           string   `json:"devops_branch"`
	TriggeredBy            string   `json:"triggered_by"`
}

// ------------------
// Stage Metrics
// ------------------

type StageMetrics struct {
	StageStatus     string  `json:"stage_status"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// ------------------
// Metrics Container
// ------------------

type MetricsInfo struct {
	PayloadType string                  `json:"payload_type"`
	StageStatus string                  `json:"stage_status"`
	Stages      map[string]StageMetrics `json:"stage"`
	JobStatus   string                  `json:"job_status"`
}

// ------------------
// Metadata
// ------------------

type MetadataInfo struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Version   string `json:"version"`
}

// ------------------
// Final Payload
// ------------------

type FusionPayload struct {
	Job                 JobInfo      `json:"job"`
	Metrics             MetricsInfo  `json:"metrics"`
	Metadata            MetadataInfo `json:"metadata"`
	OperationalCategory string       `json:"operational_category"`
}
