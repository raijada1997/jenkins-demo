# Pipeline Execution Specification - Intelligent Retry Logic

## Overview
This specification defines the behavior of pipeline execution tracking with intelligent retry detection. The system automatically infers whether an incoming build is a retry or a fresh execution based on document state analysis, eliminating reliance on manual retry flags.

---

## 1. Pipeline Stage Definitions

### 1.1 Infrastructure Stages (INSTALL)
Ordered sequence of infrastructure deployment stages:
- `pull_iso`
- `cleanup`
- `stage_1`
- `stage_2`
- `storage`
- `services`

### 1.2 Test Stages
Post-infrastructure validation stages:
- `bvt` (Build Verification Test)
- `fvt` (Functional Verification Test)
- `svt` (System Verification Test)
- `fdf_tests` (FDF Test Suite)

### 1.3 Operational Categories
- **INSTALL**: Fresh infrastructure deployment pipelines
- **UPGRADE**: Infrastructure upgrade pipelines
- **OTHER**: All other pipeline types

---

## 2. Execution Status Definitions

### 2.1 Success Definition
A pipeline execution is **successful** when:

**Option A (Full Pipeline):**
- All stages from `cleanup` → `services` have `stage_status = "success"`

**Option B (Partial Pipeline):**
- All stages from `stage_2` → `services` have `stage_status = "success"`
- (`pull_iso` and `cleanup` may be skipped)

**Additional Requirements:**
- `duration_seconds > 0` for each successful stage (ensures actual execution)
- No stage has `stage_status = "failed"`

### 2.2 Failure Definition
A pipeline execution is **failed** when:
- Any stage between `cleanup` → `services` has `stage_status = "failed"`
- OR any test stage has `stage_status = "failed"`

### 2.3 In-Progress Definition
A pipeline execution is **in-progress** when:
- At least one stage has `stage_status = "in-progress"`
- OR the pipeline has not reached the `stop_stage` yet
- AND no stages have failed

### 2.4 Skipped Status
Stages are marked as **skipped** when:
- They occur before `start_stage` in a fresh execution
- The pipeline was configured to skip them (e.g., `skip_bvt = true`)

---

## 3. Intelligent Retry Detection

### 3.1 Core Principle
**Retry detection is purely data-driven.** The system analyzes:
1. Existing document state (stages, statuses, indices)
2. Incoming payload data (start_stage, stages present)
3. Infrastructure completion status

The payload's `retry` flag is **ignored** for decision-making.

### 3.2 Retry Inference Algorithm

#### Rule 1: Test Stage Detection
```
IF start_stage IN [bvt, fvt, svt, fdf_tests]:
    RETURN retry = true
```
**Rationale:** Test stages always run on existing infrastructure.

#### Rule 2: Infrastructure Completion Check
```
IF infrastructure_completed(existing_doc):
    IF start_stage == "services":
        RETURN retry = true  // Exception: re-running last infra stage
    ELSE:
        RETURN retry = false  // Infra done, this is a new build
```

**Infrastructure Completed Definition:**
- All `pull_iso` → `services` = `success`, OR
- All `stage_2` → `services` = `success`

#### Rule 3: Infrastructure Incomplete - Index Comparison
```
IF NOT infrastructure_completed(existing_doc):
    last_idx = highest_stage_index_in_existing_doc
    incoming_idx = stage_index(start_stage)
    
    IF incoming_idx >= last_idx:
        RETURN retry = true  // Continuation of same execution
    
    IF start_stage == "stage_1":
        RETURN retry = true  // Exception: stage_1 always merges
    
    RETURN retry = false  // Behind existing progress, different build
```

### 3.3 Build Linkage Check
Before retry inference, check if the build is already linked:
```
IF build_id == existing_doc.build_id:
    RETURN stage_update (not retry)

IF build_id IN existing_doc.retry_jobs:
    RETURN stage_update (not retry)
```

---

## 4. Document Merge Behavior

### 4.1 Fresh Execution (retry = false)
When creating a new document:
- Stages before `start_stage` are marked as `"skipped"`
- `retry_jobs` map is initialized as empty
- `touch_type = "fresh"`
- `retry = false`

### 4.2 Retry Execution (retry = true)
When merging into existing document:

#### 4.2.1 Retry Jobs Metadata
Add entry to `retry_jobs` map:
```json
"retry_jobs": {
  "3200": {
    "start_stage": "stage_2",
    "stop_stage": "services",
    "triggered_by": "user@example.com",
    "timestamp": "2026-05-01T19:00:00Z",
    "jenkins_build_url": "https://jenkins.example.com/job/3200",
    "fusion_version": "1.2.3"
  }
}
```

#### 4.2.2 Manually Fixed Stage Detection
Stages between the last recorded stage and `start_stage` are marked as manually fixed:

**Detection Rules:**
1. Find `last_success_idx` = highest index with `stage_status = "success"`
2. Find `last_failure_idx` = highest index with `stage_status = "failed"`
3. Find `last_inprogress_idx` = highest index with `stage_status = "in-progress"`
4. Find `start_idx` = index of incoming `start_stage`

**Marking Logic (Union of Ranges):**
- If `last_success_idx != -1`: Mark stages in range `(last_success_idx, start_idx)` (exclusive both ends)
- If `last_failure_idx != -1`: Mark stages in range `[last_failure_idx, start_idx)` (inclusive start, exclusive end)
- If `last_inprogress_idx != -1`: Mark stages in range `[last_inprogress_idx, start_idx)` (inclusive start, exclusive end)

**Exclusions:**
- Test stages (`bvt`, `fvt`, `svt`, `fdf_tests`) are **NEVER** marked as manually fixed

**Applied Changes:**
```json
{
  "stage_status": "success",
  "manually_fixed": true
}
```

#### 4.2.3 Stage Update Behavior
- **Non-test stages**: Overwrite `stage_status` and `duration_seconds`
- **Test stages**: Use nested structure keyed by `build_id`:
```json
"fvt": {
  "2144": {
    "stage_status": "success",
    "duration_seconds": 345
  },
  "2154": {
    "stage_status": "failed",
    "duration_seconds": 120
  }
}
```

### 4.3 Stage Update (same execution)
When the same `build_id` sends subsequent payloads:
- Update stages with new data
- Do NOT add to `retry_jobs`
- `touch_type = "stage_update"`

---

## 5. Special Cases and Edge Conditions

### 5.1 Services-Only Build Rejection
**Condition:**
- Existing doc has `services` stage completed
- Existing doc has any test stage present
- Incoming build: `start_stage = "services"` AND `stop_stage = "services"`

**Action:** Reject the payload
**Rationale:** Prevents duplicate services-only runs when tests already exist

### 5.2 Stage_1 Exception
**Rule:** If `start_stage = "stage_1"`, always treat as retry (merge into existing doc)
**Rationale:** Stage_1 is often re-run to fix early failures; should update the same execution

### 5.3 Empty Stages Payload
**Rule:** Reject all payloads with `len(stages) == 0`
**Rationale:** No data to store; prevents empty documents

### 5.4 Test Stage Nested Structure Migration
When a test stage exists in old flat structure:
```json
"fvt": {
  "stage_status": "success",
  "duration_seconds": 300
}
```

Convert to nested structure:
```json
"fvt": {
  "legacy": {
    "stage_status": "success",
    "duration_seconds": 300
  },
  "2154": {
    "stage_status": "success",
    "duration_seconds": 345
  }
}
```

---

## 6. Document Fields

### 6.1 Core Fields
- `execution_key`: Unique identifier = `pipeline_name_rack_name_build_id`
- `build_id`: Primary Jenkins build ID
- `pipeline_name`: Pipeline job name
- `rack_name`: Target rack identifier
- `operational_category`: INSTALL | UPGRADE | OTHER

### 6.2 Retry Tracking Fields
- `retry`: Boolean (always `false` for new docs, not used for inference)
- `retry_jobs`: Map of build_id → retry metadata
- `touch_type`: "fresh" | "retry" | "stage_update"
- `retry_metadata`: Optional metadata for retry executions

### 6.3 Stage Fields
- `stages`: Map of stage_name → stage data
  - For non-test stages: `{stage_status, duration_seconds, manually_fixed?}`
  - For test stages: `{build_id: {stage_status, duration_seconds}}`

### 6.4 Status Fields
- `job_status`: Calculated from stage results ("success" | "failed" | "in-progress")
- `start_stage`: First stage in execution
- `stop_stage`: Last stage in execution

---

## 7. Metrics and Calculations

### 7.1 Attempt Count
```
attempts = count(all documents for rack + pipeline + operational_category)
```
Includes: Success + Failed + In-Progress

### 7.2 No-Touch Execution
A pipeline is **No-Touch** when:
- `retry_jobs` map is empty (no retries)
- All stages `cleanup` → `services` have `stage_status = "success"`
- `touch_type = "fresh"`

### 7.3 Success Rate
```
success_rate = (successful_executions / total_attempts) * 100
```

### 7.4 Retry Rate
```
retry_rate = (executions_with_retries / total_attempts) * 100
where executions_with_retries = count(docs where len(retry_jobs) > 0)
```

---

## 8. Workflow Examples

### Example 1: Fresh Install
```
Incoming: build_id=1000, start_stage=pull_iso, stages=[pull_iso, cleanup, stage_1]
Existing: None

Decision: Create new document
Result: 
  - retry = false
  - touch_type = "fresh"
  - stages: pull_iso, cleanup, stage_1 with actual data
```

### Example 2: Retry After Failure
```
Incoming: build_id=1001, start_stage=stage_2, stages=[stage_2, storage]
Existing: build_id=1000, stages={stage_1: failed}

Decision: retry = true (infra incomplete, incoming_idx > last_idx)
Result:
  - Merge into existing doc
  - Mark stage_1 as manually_fixed=true, status=success
  - Add build_id=1001 to retry_jobs
  - Update stage_2, storage with new data
  - touch_type = "retry"
```

### Example 3: Test Stage Execution
```
Incoming: build_id=1002, start_stage=bvt, stages=[bvt]
Existing: build_id=1000, infra complete

Decision: retry = true (test stage rule)
Result:
  - Merge into existing doc
  - Add bvt data under nested structure: bvt.1002
  - Add build_id=1002 to retry_jobs
  - touch_type = "retry"
```

### Example 4: Stage Update (Same Build)
```
Incoming: build_id=1000, stages=[services]
Existing: build_id=1000, stages={stage_1: success, stage_2: in-progress}

Decision: stage_update (build already linked)
Result:
  - Update existing doc
  - Update services stage
  - Do NOT add to retry_jobs
  - touch_type = "stage_update"
```

---

## 9. Implementation Notes

### 9.1 Idempotency
- Multiple payloads from the same build_id are idempotent
- Retry job entries are added only once per build_id
- Stage updates overwrite previous values

### 9.2 Timestamp Handling
- `timestamp` field always reflects the latest update
- Individual retry_jobs entries preserve their original timestamps

### 9.3 Job Status Calculation
Recalculated on every merge:
1. Check for any failed stages → `job_status = "failed"`
2. Check if `stop_stage` reached with success → `job_status = "success"`
3. Otherwise → `job_status = "in-progress"`

### 9.4 Backward Compatibility
- Old documents without `retry_jobs` are handled gracefully
- Flat test stage structures are migrated to nested format on first update
- Missing fields default to appropriate values

---

## 10. Validation Rules

### 10.1 Required Fields
- `rack_name` (must not be empty)
- `pipeline_name` (must not be empty)
- `jenkins_build_id` (must not be empty)
- `stages` (must contain at least one stage)

### 10.2 Stage Status Values
Normalized to:
- `"success"`
- `"failed"`
- `"in-progress"`
- `"skipped"`

### 10.3 Operational Category Detection
- Contains "upgrade" → UPGRADE
- Contains "hci-rack-automation" → INSTALL
- Otherwise → OTHER

---

## Document Version
**Version:** 2.0  
**Last Updated:** 2026-05-01  
**Status:** Active - Intelligent Retry Logic Implementation
