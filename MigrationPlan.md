# Migration Plan: Groovy to Python for Metrics Handling

## Overview

Based on the architecture diagram, we're migrating from **direct Groovy → Metrics** (old path) to **Groovy → Python Utility → Metrics** (blue path).

---

## Current Architecture (Old - Black Path)

```
┌─────────┐
│ Groovy  │
│         │
└────┬────┘
     │
     │ Direct POST call
     │ (httpRequest in Groovy)
     ↓
┌─────────┐
│ Metrics │
│ Server  │
└─────────┘
```

**Problems**:
- All logic in Groovy (hard to test independently)
- Difficult to reuse across different pipelines
- Limited Python ecosystem benefits
- Hard to maintain complex data transformations

---

## New Architecture (Blue Path)

```
┌─────────────────────────────────────────────────────────────┐
│                                                               │
│  ┌─────────┐                                                 │
│  │ Groovy  │                                                 │
│  │         │                                                 │
│  └────┬────┘                                                 │
│       │                                                      │
│       │ 1. Pass Build Info                                  │
│       │    (rack_name, build_id, etc.)                      │
│       ↓                                                      │
│  ┌──────────────────┐                                       │
│  │ Python Utility   │                                       │
│  │ (to be created)  │                                       │
│  └────┬─────────────┘                                       │
│       │                                                      │
│       │ 2. Fetch rack configuration                         │
│       │    from rack_configuration.json                     │
│       ↓                                                      │
│  ┌──────────────────┐                                       │
│  │ rack_config.json │                                       │
│  └────┬─────────────┘                                       │
│       │                                                      │
│       │ 3. Build Final Payload                              │
│       │    (merge build info + rack config)                 │
│       ↓                                                      │
│  ┌──────────────────┐         4. POST Request              │
│  │ Final Payload    │────────────────────────────────────►  │
│  └──────────────────┘                                       │
│                                                              │
└──────────────────────────────────────────────────────────┬──┘
                                                            │
                                                            ↓
                                                    ┌─────────┐
                                                    │ Metrics │
                                                    │ Server  │
                                                    └─────────┘
```

**Benefits**:
- Separation of concerns (Groovy for orchestration, Python for logic)
- Easier to test Python code independently
- Can fetch additional data from configuration files
- Reusable Python utility across multiple pipelines
- Better error handling and logging in Python
- Access to Python's rich ecosystem (requests, pandas, etc.)

---

## Migration Strategy

### Phase 1: Create Python Utility Structure

**Directory Structure**:
```
rack-automation/
├── jenkins/
│   └── HCI_Install.Jenkinsfile
├── python_utils/
│   ├── __init__.py
│   ├── metrics_handler.py      # Main metrics logic
│   ├── config_loader.py         # Load rack configurations
│   ├── payload_builder.py       # Build metric payloads
│   └── http_client.py           # HTTP request handling
├── config/
│   └── rack_configuration.json  # Rack metadata
├── tests/
│   ├── test_metrics_handler.py
│   └── test_payload_builder.py
└── requirements.txt
```

### Phase 2: Implement Python Modules

#### Module 1: `config_loader.py`
**Purpose**: Load rack configuration from JSON file

```python
import json
from typing import Dict, Optional

class RackConfigLoader:
    """Loads rack configuration from rack_configuration.json"""
    
    def __init__(self, config_path: str = "config/rack_configuration.json"):
        self.config_path = config_path
        self.config_data = self._load_config()
    
    def _load_config(self) -> Dict:
        """Load the configuration file"""
        try:
            with open(self.config_path, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"Warning: Config file not found at {self.config_path}")
            return {}
        except json.JSONDecodeError as e:
            print(f"Error parsing JSON: {e}")
            return {}
    
    def get_rack_info(self, rack_name: str) -> Optional[Dict]:
        """
        Get configuration for a specific rack
        
        Args:
            rack_name: Name of the rack (e.g., "rackm03")
        
        Returns:
            Dictionary with rack configuration or None if not found
        """
        return self.config_data.get(rack_name)
    
    def get_all_racks(self) -> Dict:
        """Get all rack configurations"""
        return self.config_data
```

**Example rack_configuration.json**:
```json
{
  "rackm03": {
    "location": "RTP",
    "datacenter": "Building 5",
    "hardware_type": "LENOVO",
    "network_zone": "zone-a",
    "owner_team": "fusion-hci",
    "capacity": {
      "nodes": 6,
      "storage_tb": 100
    }
  },
  "rackl": {
    "location": "RCH",
    "datacenter": "Building 2",
    "hardware_type": "DELL",
    "network_zone": "zone-b",
    "owner_team": "fusion-hci",
    "capacity": {
      "nodes": 3,
      "storage_tb": 50
    }
  }
}
```

---

#### Module 2: `payload_builder.py`
**Purpose**: Build the metrics payload by combining build info and rack config

```python
from typing import Dict, List, Optional
from datetime import datetime, timezone

class MetricsPayloadBuilder:
    """Builds metrics payload for sending to metrics server"""
    
    def __init__(self):
        self.payload = {}
    
    def build_job_info(self, build_params: Dict) -> Dict:
        """
        Build job information section
        
        Args:
            build_params: Dictionary with build parameters from Jenkins
        
        Returns:
            Dictionary with job information
        """
        return {
            "pipeline_name": build_params.get("pipeline_name", "hci-rack-automation-install"),
            "rack_name": build_params.get("rack_name", ""),
            "rack_ip": build_params.get("rack_ip", ""),
            "start_stage": build_params.get("start_stage", ""),
            "stop_stage": build_params.get("stop_stage", ""),
            "fusion_version": build_params.get("fusion_version", ""),
            "ocp_version": build_params.get("ocp_version", ""),
            "environment": build_params.get("environment", "Dev"),
            "lab_location": build_params.get("lab_location", ""),
            "install_type": build_params.get("install_type", "online"),
            "storage_type": build_params.get("storage_type", ""),
            "retry": build_params.get("retry", False),
            "jenkins_build_id": build_params.get("jenkins_build_id", ""),
            "jenkins_build_url": build_params.get("jenkins_build_url", ""),
            "isf_operator_build_id": build_params.get("isf_operator_build_id", ""),
            "triggered_by": build_params.get("triggered_by", "automated"),
            # Add rack configuration data
            "rack_metadata": build_params.get("rack_metadata", {})
        }
    
    def build_stage_metrics(self, stage_name: str, status: str, duration: int = 0) -> Dict:
        """
        Build stage-level metrics
        
        Args:
            stage_name: Name of the stage
            status: Status (In-progress, success, failure, skipped)
            duration: Duration in seconds
        
        Returns:
            Dictionary with stage metrics
        """
        return {
            "payload_type": "stage",
            "stage_status": status,
            "stage": {
                stage_name: {
                    "stage_status": status,
                    "duration_seconds": duration
                }
            },
            "job_status": ""
        }
    
    def build_job_metrics(self, all_stages: Dict, job_status: str) -> Dict:
        """
        Build job-level metrics with all stages
        
        Args:
            all_stages: Dictionary of all stage metrics
            job_status: Final job status (SUCCESS, FAILURE, etc.)
        
        Returns:
            Dictionary with job metrics
        """
        return {
            "payload_type": "job",
            "stage_status": "",
            "stage": all_stages,
            "job_status": job_status
        }
    
    def build_complete_payload(
        self,
        job_info: Dict,
        metrics: Dict,
        metric_type: str = "stage"
    ) -> Dict:
        """
        Build the complete payload
        
        Args:
            job_info: Job information dictionary
            metrics: Metrics dictionary (stage or job)
            metric_type: Type of metric ('stage' or 'job')
        
        Returns:
            Complete payload ready to send
        """
        return {
            "job": job_info,
            "metrics": metrics,
            "metadata": {
                "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
                "source": "jenkins",
                "version": "v1",
                "metric_type": metric_type
            }
        }
```

---

#### Module 3: `http_client.py`
**Purpose**: Handle HTTP requests to metrics server

```python
import requests
import json
from typing import Dict, Optional
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MetricsHTTPClient:
    """HTTP client for sending metrics to the metrics server"""
    
    def __init__(
        self,
        endpoint_url: str = "https://jenkins-metric-prod.apps.inst-metrics.fusion.tadn.ibm.com/fusion-metrics",
        timeout: int = 30
    ):
        self.endpoint_url = endpoint_url
        self.timeout = timeout
    
    def send_metrics(self, payload: Dict) -> bool:
        """
        Send metrics payload to the server
        
        Args:
            payload: Complete metrics payload
        
        Returns:
            True if successful, False otherwise
        """
        try:
            logger.info(f"Sending metrics to {self.endpoint_url}")
            logger.debug(f"Payload: {json.dumps(payload, indent=2)}")
            
            response = requests.post(
                self.endpoint_url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=self.timeout,
                verify=False  # Ignore SSL errors for internal servers
            )
            
            response.raise_for_status()
            
            logger.info(f"✓ Metrics sent successfully. Status: {response.status_code}")
            return True
            
        except requests.exceptions.Timeout:
            logger.error(f"✗ Timeout sending metrics (>{self.timeout}s)")
            return False
            
        except requests.exceptions.RequestException as e:
            logger.error(f"✗ Error sending metrics: {e}")
            return False
        
        except Exception as e:
            logger.error(f"✗ Unexpected error: {e}")
            return False
```

---

#### Module 4: `metrics_handler.py` (Main Entry Point)
**Purpose**: Main orchestrator that ties everything together

```python
import sys
import json
import argparse
from typing import Dict, Optional
from config_loader import RackConfigLoader
from payload_builder import MetricsPayloadBuilder
from http_client import MetricsHTTPClient

class MetricsHandler:
    """Main handler for metrics operations"""
    
    def __init__(self, config_path: str = "config/rack_configuration.json"):
        self.config_loader = RackConfigLoader(config_path)
        self.payload_builder = MetricsPayloadBuilder()
        self.http_client = MetricsHTTPClient()
    
    def send_stage_metric(
        self,
        build_params: Dict,
        stage_name: str,
        status: str,
        duration: int = 0
    ) -> bool:
        """
        Send stage-level metrics
        
        Args:
            build_params: Build parameters from Jenkins
            stage_name: Name of the stage
            status: Stage status
            duration: Duration in seconds
        
        Returns:
            True if successful, False otherwise
        """
        # Enrich build params with rack configuration
        rack_name = build_params.get("rack_name")
        if rack_name:
            rack_config = self.config_loader.get_rack_info(rack_name)
            if rack_config:
                build_params["rack_metadata"] = rack_config
        
        # Build payload
        job_info = self.payload_builder.build_job_info(build_params)
        stage_metrics = self.payload_builder.build_stage_metrics(
            stage_name, status, duration
        )
        payload = self.payload_builder.build_complete_payload(
            job_info, stage_metrics, "stage"
        )
        
        # Send to server
        return self.http_client.send_metrics(payload)
    
    def send_job_metric(
        self,
        build_params: Dict,
        all_stages: Dict,
        job_status: str
    ) -> bool:
        """
        Send job-level metrics
        
        Args:
            build_params: Build parameters from Jenkins
            all_stages: Dictionary of all stage metrics
            job_status: Final job status
        
        Returns:
            True if successful, False otherwise
        """
        # Enrich build params with rack configuration
        rack_name = build_params.get("rack_name")
        if rack_name:
            rack_config = self.config_loader.get_rack_info(rack_name)
            if rack_config:
                build_params["rack_metadata"] = rack_config
        
        # Build payload
        job_info = self.payload_builder.build_job_info(build_params)
        job_metrics = self.payload_builder.build_job_metrics(all_stages, job_status)
        payload = self.payload_builder.build_complete_payload(
            job_info, job_metrics, "job"
        )
        
        # Send to server
        return self.http_client.send_metrics(payload)


def main():
    """CLI entry point for the metrics handler"""
    parser = argparse.ArgumentParser(description="Send metrics to metrics server")
    parser.add_argument(
        "--type",
        choices=["stage", "job"],
        required=True,
        help="Type of metric to send"
    )
    parser.add_argument(
        "--build-params",
        required=True,
        help="JSON string with build parameters"
    )
    parser.add_argument(
        "--stage-name",
        help="Stage name (required for stage metrics)"
    )
    parser.add_argument(
        "--status",
        required=True,
        help="Status (In-progress, success, failure, skipped, SUCCESS, FAILURE)"
    )
    parser.add_argument(
        "--duration",
        type=int,
        default=0,
        help="Duration in seconds"
    )
    parser.add_argument(
        "--all-stages",
        help="JSON string with all stages (required for job metrics)"
    )
    parser.add_argument(
        "--config-path",
        default="config/rack_configuration.json",
        help="Path to rack configuration file"
    )
    
    args = parser.parse_args()
    
    # Parse build parameters
    try:
        build_params = json.loads(args.build_params)
    except json.JSONDecodeError as e:
        print(f"Error parsing build-params JSON: {e}")
        sys.exit(1)
    
    # Initialize handler
    handler = MetricsHandler(args.config_path)
    
    # Send metrics based on type
    if args.type == "stage":
        if not args.stage_name:
            print("Error: --stage-name is required for stage metrics")
            sys.exit(1)
        
        success = handler.send_stage_metric(
            build_params,
            args.stage_name,
            args.status,
            args.duration
        )
    else:  # job
        if not args.all_stages:
            print("Error: --all-stages is required for job metrics")
            sys.exit(1)
        
        try:
            all_stages = json.loads(args.all_stages)
        except json.JSONDecodeError as e:
            print(f"Error parsing all-stages JSON: {e}")
            sys.exit(1)
        
        success = handler.send_job_metric(
            build_params,
            all_stages,
            args.status
        )
    
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
```

---

### Phase 3: Modify Groovy Code

**Before (Old Groovy Code)**:
```groovy
def sendMetric(String metricType, String stageName = null, String status = null, Integer duration = null) {
    try {
        // Build job info
        def jobInfo = buildJobInfo()
        
        // Build metrics
        def metricsSection = [:]
        if (metricType == 'stage') {
            metricsSection = [
                payload_type: "stage",
                stage_status: status,
                stage: [(stageName): [stage_status: status, duration_seconds: duration]]
            ]
        }
        
        // Build payload
        def payload = [job: jobInfo, metrics: metricsSection, metadata: [...]]
        
        // Send HTTP request
        def jsonPayload = groovy.json.JsonOutput.toJson(payload)
        httpRequest(
            httpMode: 'POST',
            contentType: 'APPLICATION_JSON',
            requestBody: jsonPayload,
            url: endpointUrl,
            timeout: 30
        )
    } catch (Exception e) {
        echo "Error sending metrics: ${e.getMessage()}"
    }
}
```

**After (New Groovy Code - Calls Python)**:
```groovy
def sendMetric(String metricType, String stageName = null, String status = null, Integer duration = null) {
    try {
        // Build job info (keep this in Groovy as it uses Jenkins context)
        def jobInfo = buildJobInfo()
        
        // Convert to JSON string for passing to Python
        def buildParamsJson = groovy.json.JsonOutput.toJson(jobInfo)
        
        // Build Python command
        def pythonCmd = "python3 python_utils/metrics_handler.py"
        pythonCmd += " --type ${metricType}"
        pythonCmd += " --build-params '${buildParamsJson}'"
        pythonCmd += " --status '${status}'"
        
        if (metricType == 'stage') {
            pythonCmd += " --stage-name '${stageName}'"
            pythonCmd += " --duration ${duration ?: 0}"
        } else {
            // For job metrics, pass all accumulated stages
            def allStagesJson = groovy.json.JsonOutput.toJson(stageMetrics)
            pythonCmd += " --all-stages '${allStagesJson}'"
        }
        
        // Execute Python script
        def result = sh(script: pythonCmd, returnStatus: true)
        
        if (result == 0) {
            echo "[Metrics] Successfully sent ${metricType} metrics${stageName ? ' for stage: ' + stageName : ''}"
        } else {
            echo "[Metrics] Failed to send ${metricType} metrics (exit code: ${result})"
        }
        
    } catch (Exception e) {
        echo "[Metrics] ERROR: Failed to send ${metricType} metrics"
        echo "[Metrics] Error details: ${e.getMessage()}"
        // Don't fail the build due to metrics errors
    }
}
```

---

## Complete Example: Stage Metric Flow

### Step 1: Groovy Stage Execution

```groovy
stage("Pull ISO") {
    startStageTracking("pull_iso")
    try {
        // Send in-progress metric
        sendMetric('stage', 'pull_iso', 'In-progress')
        
        // Do actual work
        sh "./pull_iso.sh ${isoBuildId}"
        
        // Send success metric
        sendMetric('stage', 'pull_iso', 'success')
        
    } catch (Exception e) {
        // Send failure metric
        sendMetric('stage', 'pull_iso', 'failure')
        throw e
    }
}
```

### Step 2: Groovy Calls Python

```groovy
def sendMetric(String metricType, String stageName, String status) {
    def jobInfo = [
        rack_name: "rackm03",
        rack_ip: "9.42.56.43",
        fusion_version: "2.8.0",
        jenkins_build_id: "456",
        // ... other fields
    ]
    
    def buildParamsJson = groovy.json.JsonOutput.toJson(jobInfo)
    
    sh """
        python3 python_utils/metrics_handler.py \
            --type stage \
            --build-params '${buildParamsJson}' \
            --stage-name 'pull_iso' \
            --status 'success' \
            --duration 1200
    """
}
```

### Step 3: Python Processes Request

```python
# metrics_handler.py receives:
# --type stage
# --build-params '{"rack_name": "rackm03", ...}'
# --stage-name 'pull_iso'
# --status 'success'
# --duration 1200

handler = MetricsHandler()

# 1. Load rack configuration
rack_config = config_loader.get_rack_info("rackm03")
# Returns: {"location": "RTP", "hardware_type": "LENOVO", ...}

# 2. Merge build params with rack config
build_params["rack_metadata"] = rack_config

# 3. Build payload
job_info = {
    "rack_name": "rackm03",
    "rack_ip": "9.42.56.43",
    "fusion_version": "2.8.0",
    "rack_metadata": {
        "location": "RTP",
        "hardware_type": "LENOVO",
        "datacenter": "Building 5"
    }
}

stage_metrics = {
    "payload_type": "stage",
    "stage_status": "success",
    "stage": {
        "pull_iso": {
            "stage_status": "success",
            "duration_seconds": 1200
        }
    }
}

payload = {
    "job": job_info,
    "metrics": stage_metrics,
    "metadata": {
        "timestamp": "2026-05-01T12:00:00Z",
        "source": "jenkins",
        "version": "v1"
    }
}

# 4. Send to metrics server
http_client.send_metrics(payload)
```

### Step 4: Final Payload Sent to Metrics Server

```json
{
  "job": {
    "pipeline_name": "hci-rack-automation-install",
    "rack_name": "rackm03",
    "rack_ip": "9.42.56.43",
    "fusion_version": "2.8.0",
    "ocp_version": "4.16",
    "jenkins_build_id": "456",
    "rack_metadata": {
      "location": "RTP",
      "datacenter": "Building 5",
      "hardware_type": "LENOVO",
      "network_zone": "zone-a",
      "owner_team": "fusion-hci"
    }
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
    "timestamp": "2026-05-01T12:00:00Z",
    "source": "jenkins",
    "version": "v1",
    "metric_type": "stage"
  }
}
```

---

## Benefits of This Approach

### 1. **Separation of Concerns**
- **Groovy**: Orchestration, Jenkins context, pipeline flow
- **Python**: Business logic, data processing, HTTP requests

### 2. **Enhanced Data**
- Automatically enriches metrics with rack configuration
- No need to pass everything from Groovy
- Centralized configuration management

### 3. **Testability**
```python
# Easy to test Python code independently
def test_payload_builder():
    builder = MetricsPayloadBuilder()
    payload = builder.build_stage_metrics("test_stage", "success", 100)
    assert payload["stage"]["test_stage"]["duration_seconds"] == 100
```

### 4. **Reusability**
- Same Python utility can be used by multiple Jenkinsfiles
- Can be called from other automation scripts
- Can be used for local testing

### 5. **Better Error Handling**
```python
try:
    response = requests.post(url, json=payload)
    response.raise_for_status()
except requests.exceptions.Timeout:
    logger.error("Timeout - retry logic here")
except requests.exceptions.HTTPError as e:
    logger.error(f"HTTP error: {e.response.status_code}")
```

### 6. **Easier Maintenance**
- Python code is easier to read and modify
- Can add features without touching Groovy
- Better logging and debugging

---

## Migration Checklist

- [ ] Create `python_utils/` directory structure
- [ ] Implement `config_loader.py`
- [ ] Implement `payload_builder.py`
- [ ] Implement `http_client.py`
- [ ] Implement `metrics_handler.py` with CLI
- [ ] Create `config/rack_configuration.json`
- [ ] Write unit tests for Python modules
- [ ] Update `sendMetric()` in Jenkinsfile to call Python
- [ ] Update `sendMetrics()` wrapper function
- [ ] Update `sendJobMetrics()` wrapper function
- [ ] Test with a single stage first
- [ ] Test with complete pipeline
- [ ] Update documentation
- [ ] Deploy to production

---

## Testing Strategy

### Unit Tests (Python)
```python
# tests/test_payload_builder.py
import pytest
from python_utils.payload_builder import MetricsPayloadBuilder

def test_build_stage_metrics():
    builder = MetricsPayloadBuilder()
    result = builder.build_stage_metrics("test_stage", "success", 100)
    
    assert result["payload_type"] == "stage"
    assert result["stage_status"] == "success"
    assert result["stage"]["test_stage"]["duration_seconds"] == 100

def test_build_job_info():
    builder = MetricsPayloadBuilder()
    params = {
        "rack_name": "test_rack",
        "fusion_version": "2.8.0"
    }
    result = builder.build_job_info(params)
    
    assert result["rack_name"] == "test_rack"
    assert result["fusion_version"] == "2.8.0"
```

### Integration Tests (Groovy + Python)
```groovy
// Test in Jenkins
stage("Test Metrics") {
    def testParams = [
        rack_name: "rackm03",
        fusion_version: "2.8.0"
    ]
    
    // Test stage metric
    sendMetric('stage', 'test_stage', 'success', 100)
    
    // Verify it was sent (check logs)
    echo "Metrics test completed"
}
```

---

## Rollback Plan

If issues arise:

1. **Keep old Groovy code** in a separate function:
   ```groovy
   def sendMetricOld(...)  // Original implementation
   def sendMetric(...)     // New Python-based implementation
   ```

2. **Feature flag** to switch between implementations:
   ```groovy
   def USE_PYTHON_METRICS = env.USE_PYTHON_METRICS ?: 'false'
   
   if (USE_PYTHON_METRICS == 'true') {
       sendMetricPython(...)
   } else {
       sendMetricOld(...)
   }
   ```

3. **Gradual rollout**:
   - Week 1: Test on dev racks only
   - Week 2: Enable for 25% of production runs
   - Week 3: Enable for 50% of production runs
   - Week 4: Enable for 100% of production runs

---

## Next Steps

1. **Review this plan** with the team
2. **Create Python utility structure** (Phase 1)
3. **Implement Python modules** (Phase 2)
4. **Test Python modules independently**
5. **Modify Groovy code** (Phase 3)
6. **Integration testing**
7. **Deploy to production**

Would you like me to start implementing any specific module?
