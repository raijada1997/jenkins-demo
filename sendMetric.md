# Deep Dive: sendMetric() Function Explained

## Overview

The [`sendMetric()`](jenkins/HCI_Install.Jenkinsfile:2134) function is a telemetry/monitoring system that sends performance and status data to a centralized metrics server. Think of it as a "health reporter" that tells a monitoring dashboard what's happening in your pipeline.

---

## Function Signature

```groovy
def sendMetric(String metricType, String stageName = null, String status = null, Integer duration = null)
```

### Parameters:
- **metricType**: Either `'stage'` or `'job'`
  - `'stage'`: Reports on individual pipeline stages (e.g., "Pull ISO", "Stage 1")
  - `'job'`: Reports on the entire pipeline run
- **stageName**: Name of the stage (required for stage metrics, e.g., "pull_iso", "Cleanup")
- **status**: Current status (e.g., "In-progress", "success", "failure", "skipped")
- **duration**: How long the stage took in seconds (optional, auto-calculated if not provided)

---

## What Happens Step-by-Step

### Step 1: Validation (Lines 2136-2139)

```groovy
if (metricType != 'stage' && metricType != 'job') {
    throw new IllegalArgumentException("Invalid metricType: ${metricType}. Must be 'stage' or 'job'")
}
```

**Purpose**: Ensures only valid metric types are used
**Action**: Throws error if metricType is not 'stage' or 'job'

---

### Step 2: Build Job Information (Line 2142)

```groovy
def jobInfo = buildJobInfo()
```

**Purpose**: Collects all the context about this pipeline run

**What [`buildJobInfo()`](jenkins/HCI_Install.Jenkinsfile:2036) returns** (Lines 2036-2118):

```groovy
{
    pipeline_name: "hci-rack-automation-install",
    rack_name: "rackm03",
    rack_ip: "9.42.56.43",
    start_stage: "Cleanup",
    stop_stage: "Services",
    fusion_version: "2.8.0",
    ocp_version: "4.16",
    environment: "Dev",
    lab_location: "RTP",
    install_type: "online",           // or "offline"
    storage_type: "Scale",
    retry: false,                     // Is this a retry?
    jenkins_build_id: "123",
    jenkins_build_url: "https://jenkins.example.com/job/123",
    isf_operator_build_id: "2.8.0-11904029-linux.amd64",
    topology: "Standalone",
    rack_type: "LENOVO",
    is_production: false,
    services_to_install: ["Guardian", "Discover"],
    triggered_by: "john.doe@ibm.com",
    // ... and many more fields
}
```

**Think of it as**: A "business card" for this pipeline run with all identifying information

---

### Step 3: Build Metrics Section (Lines 2144-2191)

This is where it gets interesting - the function behaves differently based on `metricType`:

#### Option A: Stage Metrics (Lines 2147-2175)

```groovy
if (metricType == 'stage') {
    // Stage-level metrics
```

**Purpose**: Report on a single stage's performance

**Process**:

1. **Validate stageName** (Lines 2149-2151):
   ```groovy
   if (!stageName) {
       throw new IllegalArgumentException("stageName is required for stage metrics")
   }
   ```

2. **Calculate Duration** (Lines 2154-2161):
   ```groovy
   def stageDuration = duration
   if (stageDuration == null) {
       if (status == "In-progress" || status?.toLowerCase() == "skipped") {
           stageDuration = 0  // No duration yet for in-progress/skipped
       } else {
           stageDuration = stopStageTracking(stageName, status ?: "Unknown")
       }
   }
   ```
   
   **What this means**:
   - If duration is provided, use it
   - If stage is "In-progress" or "skipped", duration = 0
   - Otherwise, call [`stopStageTracking()`](jenkins/HCI_Install.Jenkinsfile:1997) which:
     - Calculates: `endTime - startTime`
     - Returns duration in seconds
     - Stores the result for later use

3. **Build Stage Entry** (Lines 2164-2168):
   ```groovy
   def currentStageEntry = [:]
   currentStageEntry[stageName] = [
       stage_status: status ?: "Unknown",
       duration_seconds: stageDuration
   ]
   ```
   
   **Example**:
   ```groovy
   {
       "pull_iso": {
           "stage_status": "success",
           "duration_seconds": 1200  // 20 minutes
       }
   }
   ```

4. **Build Metrics Section** (Lines 2170-2175):
   ```groovy
   metricsSection = [
       payload_type: "stage",
       stage_status: status ?: "Unknown",
       stage: currentStageEntry,
       job_status: ""  // Empty for stage metrics
   ]
   ```

#### Option B: Job Metrics (Lines 2176-2191)

```groovy
else {
    // Job-level metrics with all accumulated stages
```

**Purpose**: Report on the entire pipeline run with all stages

**Process**:

1. **Get Final Job Status** (Line 2178):
   ```groovy
   def jobStatus = status ?: getFinalBuildStatus()
   ```
   - Uses provided status OR
   - Calls [`getFinalBuildStatus()`](jenkins/HCI_Install.Jenkinsfile:2022) which checks:
     - `currentBuild.result` (explicit result)
     - `currentBuild.currentResult` (current state)
     - Returns: "SUCCESS", "FAILURE", "UNSTABLE", etc.

2. **Build Metrics Section** (Lines 2180-2190):
   ```groovy
   metricsSection = [
       payload_type: "job",
       stage_status: "",  // Empty for job metrics
       stage: stageMetrics.clone(),  // ALL stages accumulated
       job_status: jobStatus
   ]
   ```
   
   **What is `stageMetrics`?**
   - A global map (defined at line 1980) that accumulates ALL stage data
   - Example:
   ```groovy
   {
       "pull_iso": {
           "stage_status": "success",
           "duration_seconds": 1200
       },
       "Cleanup": {
           "stage_status": "success",
           "duration_seconds": 3600
       },
       "stage_1": {
           "stage_status": "success",
           "duration_seconds": 2400
       }
       // ... all other stages
   }
   ```

---

### Step 4: Build Complete Payload (Lines 2194-2202)

```groovy
def payload = [
    job: jobInfo,           // All job context
    metrics: metricsSection, // Stage or job metrics
    metadata: [
        timestamp: new Date().format("yyyy-MM-dd'T'HH:mm:ss'Z'", TimeZone.getTimeZone("UTC")),
        source: "jenkins",
        version: "v1"
    ]
]
```

**Complete Payload Example** (Stage Metric):

```json
{
  "job": {
    "pipeline_name": "hci-rack-automation-install",
    "rack_name": "rackm03",
    "rack_ip": "9.42.56.43",
    "fusion_version": "2.8.0",
    "ocp_version": "4.16",
    "jenkins_build_id": "123",
    "triggered_by": "john.doe@ibm.com"
    // ... 30+ more fields
  },
  "metrics": {
    "payload_type": "stage",
    "stage_status": "success",
    "stage": {
      "pull_iso": {
        "stage_status": "success",
        "duration_seconds": 1200
      }
    },
    "job_status": ""
  },
  "metadata": {
    "timestamp": "2026-05-01T09:00:00Z",
    "source": "jenkins",
    "version": "v1"
  }
}
```

**Complete Payload Example** (Job Metric):

```json
{
  "job": {
    "pipeline_name": "hci-rack-automation-install",
    "rack_name": "rackm03",
    // ... all job info
  },
  "metrics": {
    "payload_type": "job",
    "stage_status": "",
    "stage": {
      "pull_iso": {
        "stage_status": "success",
        "duration_seconds": 1200
      },
      "Cleanup": {
        "stage_status": "success",
        "duration_seconds": 3600
      },
      "stage_1": {
        "stage_status": "success",
        "duration_seconds": 2400
      }
      // ... all stages
    },
    "job_status": "SUCCESS"
  },
  "metadata": {
    "timestamp": "2026-05-01T09:00:00Z",
    "source": "jenkins",
    "version": "v1"
  }
}
```

---

### Step 5: Convert to JSON (Lines 2205-2206)

```groovy
def jsonPayload = groovy.json.JsonOutput.toJson(payload)
def jsonPretty = groovy.json.JsonOutput.prettyPrint(jsonPayload)
```

**Purpose**: Convert the Groovy map to JSON string
- `jsonPayload`: Compact JSON (for sending)
- `jsonPretty`: Formatted JSON (for logging)

---

### Step 6: Log the Payload (Lines 2209-2210)

```groovy
echo "[Metrics] Sending ${metricType} metrics:"
echo jsonPretty
```

**Purpose**: Print to Jenkins console log for debugging

**Example Output**:
```
[Metrics] Sending stage metrics:
{
  "job": {
    "pipeline_name": "hci-rack-automation-install",
    ...
  },
  "metrics": {
    "payload_type": "stage",
    ...
  }
}
```

---

### Step 7: Send HTTP Request (Lines 2213-2224)

```groovy
def endpointUrl = 'https://jenkins-metric-prod.apps.inst-metrics.fusion.tadn.ibm.com/fusion-metrics'

httpRequest(
    httpMode: 'POST',
    contentType: 'APPLICATION_JSON',
    requestBody: jsonPayload,
    url: endpointUrl,
    ignoreSslErrors: true,
    validResponseCodes: '200:299',
    timeout: 30
)
```

**What this does**:
1. **Sends HTTP POST request** to the metrics server
2. **Content-Type**: `application/json`
3. **Body**: The JSON payload
4. **SSL**: Ignores SSL certificate errors (for internal servers)
5. **Success codes**: 200-299 (any 2xx response is OK)
6. **Timeout**: 30 seconds (fails if server doesn't respond)

**Think of it as**: Sending a text message to a monitoring dashboard

---

### Step 8: Success Confirmation (Line 2226)

```groovy
echo "[Metrics] Successfully sent ${metricType} metrics${stageName ? ' for stage: ' + stageName : ''}"
```

**Example Output**:
```
[Metrics] Successfully sent stage metrics for stage: pull_iso
```

---

### Step 9: Error Handling (Lines 2228-2232)

```groovy
catch (Exception e) {
    echo "[Metrics] ERROR: Failed to send ${metricType} metrics${stageName ? ' for stage: ' + stageName : ''}"
    echo "[Metrics] Error details: ${e.getMessage()}"
    // Don't fail the build due to metrics errors
}
```

**Important**: If metrics fail, the pipeline continues!
- Logs the error
- Does NOT throw exception
- Does NOT fail the build

**Why?**: Metrics are nice-to-have, not critical. The installation should continue even if metrics fail.

---

## How It's Used in the Pipeline

### Example 1: Stage Start (In-Progress)

```groovy
stage("Pull ISO") {
    startStageTracking("pull_iso")  // Start timer
    try {
        sendMetrics("pull_iso", "In-progress")  // Send in-progress metric
        // ... do work ...
        sendMetrics("pull_iso", "success")      // Send success metric
    } catch (Exception e) {
        sendMetrics("pull_iso", "failure")      // Send failure metric
    }
}
```

### Example 2: Job Completion

```groovy
// At the end of pipeline
sendJobMetrics(totalDuration, "SUCCESS")
```

This sends ALL accumulated stage metrics plus the final job status.

---

## Data Flow Diagram

```
Pipeline Stage Execution
         ↓
   startStageTracking()
   (Records start time)
         ↓
   sendMetric('stage', 'pull_iso', 'In-progress')
         ↓
   [Builds payload with job info + stage info]
         ↓
   [Converts to JSON]
         ↓
   [HTTP POST to metrics server]
         ↓
   [Metrics server stores data]
         ↓
   [Dashboard displays metrics]
         ↓
   Stage completes
         ↓
   stopStageTracking()
   (Calculates duration)
         ↓
   sendMetric('stage', 'pull_iso', 'success')
         ↓
   [Sends final stage metrics with duration]
```

---

## What the Metrics Server Does

The metrics server (at `jenkins-metric-prod.apps.inst-metrics.fusion.tadn.ibm.com`) likely:

1. **Receives** the JSON payload
2. **Stores** it in a database (probably time-series database like InfluxDB or Prometheus)
3. **Aggregates** data across multiple pipeline runs
4. **Provides** dashboards showing:
   - Average stage durations
   - Success/failure rates
   - Trends over time
   - Performance bottlenecks
   - Which racks are most problematic
   - Which stages fail most often

---

## Real-World Example

Let's say you run the pipeline:

**Stage 1: Pull ISO**
```groovy
sendMetric('stage', 'pull_iso', 'In-progress')
// ... 20 minutes later ...
sendMetric('stage', 'pull_iso', 'success')
```

**Payload sent**:
```json
{
  "job": {
    "rack_name": "rackm03",
    "jenkins_build_id": "456"
  },
  "metrics": {
    "payload_type": "stage",
    "stage_status": "success",
    "stage": {
      "pull_iso": {
        "stage_status": "success",
        "duration_seconds": 1200
      }
    }
  }
}
```

**Stage 2: Cleanup**
```groovy
sendMetric('stage', 'Cleanup', 'success')
```

**Stage 3: Stage 1**
```groovy
sendMetric('stage', 'stage_1', 'success')
```

**End of Pipeline**
```groovy
sendMetric('job', null, 'SUCCESS')
```

**Final Job Payload**:
```json
{
  "job": { ... },
  "metrics": {
    "payload_type": "job",
    "stage": {
      "pull_iso": { "stage_status": "success", "duration_seconds": 1200 },
      "Cleanup": { "stage_status": "success", "duration_seconds": 3600 },
      "stage_1": { "stage_status": "success", "duration_seconds": 2400 }
    },
    "job_status": "SUCCESS"
  }
}
```

---

## Key Takeaways

1. **Purpose**: Telemetry system for monitoring pipeline performance
2. **Two Types**: Stage metrics (individual stages) and Job metrics (entire run)
3. **Non-Blocking**: Failures don't stop the pipeline
4. **Rich Context**: Includes 30+ fields about the installation
5. **Time Tracking**: Automatically calculates stage durations
6. **Centralized**: All data goes to one metrics server
7. **Actionable**: Helps teams identify bottlenecks and failures

**In Simple Terms**: It's like a fitness tracker for your pipeline - it tracks how long each stage takes, whether it succeeded or failed, and sends all that data to a dashboard so teams can see trends and improve performance over time.
