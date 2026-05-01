# HCI Install Jenkinsfile - Complete Explanation

## Table of Contents
1. [What is Jenkins and Jenkinsfile?](#what-is-jenkins-and-jenkinsfile)
2. [File Overview](#file-overview)
3. [Key Components Breakdown](#key-components-breakdown)
4. [Pipeline Stages Explained](#pipeline-stages-explained)
5. [How Everything Works Together](#how-everything-works-together)

---

## What is Jenkins and Jenkinsfile?

### Jenkins
- **Jenkins** is an automation server used for Continuous Integration/Continuous Deployment (CI/CD)
- It automates building, testing, and deploying software
- Think of it as a robot that runs your tasks automatically

### Jenkinsfile
- A **Jenkinsfile** is a text file that defines a Jenkins Pipeline
- Written in **Groovy** (a Java-based scripting language)
- Contains all the instructions for what Jenkins should do
- This specific file automates the installation of HCI (Hyper-Converged Infrastructure) systems

---

## File Overview

**File**: `jenkins/HCI_Install.Jenkinsfile`
**Size**: 3,169 lines
**Purpose**: Automates the complete installation and testing of IBM Fusion HCI systems

### What This Pipeline Does:
1. **Prepares the environment** (downloads ISO images, sets up configurations)
2. **Installs the system** in multiple stages (Cleanup, Stage 1, Stage 2, Storage, Services)
3. **Runs tests** (BVT, FVT, SVT, FDF-TESTS)
4. **Reports results** (sends notifications, creates issues, generates metrics)

---

## Key Components Breakdown

### 1. **Library Import** (Line 1)
```groovy
@Library("DevOps-Jenkins-CommonLibrary@main") _
```
- **What it does**: Imports a shared library of reusable Jenkins functions
- **Why**: Avoids code duplication across multiple pipelines
- **Think of it as**: Importing a Python module like `import common_functions`

### 2. **Import Statements** (Lines 2-6)
```groovy
import groovy.json.JsonSlurper      // Parse JSON files
import groovy.json.JsonOutput       // Create JSON output
import org.jenkinsci.plugins.workflow.steps.FlowInterruptedException  // Handle timeouts
import org.yaml.snakeyaml.Yaml      // Parse YAML files
import groovy.transform.Field       // Create global variables
```
- These are like Python imports: `import json`, `import yaml`

### 3. **Pipeline Parameters** (Lines 8-337)

#### What are Parameters?
Parameters are inputs that users provide when running the pipeline. Think of them as function arguments.

#### Key Parameters Explained:

**a) LAB_LOCATION** (Line 10)
```groovy
choice(name: 'LAB_LOCATION', choices: ['RCH', 'RTP'])
```
- **Type**: Dropdown choice
- **Purpose**: Select which physical lab to run the installation in
- **Options**: RCH (Research Triangle Park) or RTP (Raleigh)

**b) START and STOP** (Lines 12-62)
```groovy
name: 'START'  // Which stage to start from
name: 'STOP'   // Which stage to stop at
```
- **Purpose**: Allows partial pipeline execution
- **Stages**: Cleanup → Stage 1 → Stage 2 → Storage → Services → BVT → FVT → SVT → FDF-TESTS
- **Example**: Start from "Stage 2" and stop at "Services" (skips earlier stages)

**c) LAPTOPIP_RACKNAME** (Line 63)
```groovy
string(name: 'LAPTOPIP_RACKNAME')
```
- **Format**: `9.42.56.43_rackname`
- **Purpose**: Identifies which physical rack to install on
- **Contains**: IP address + rack identifier

**d) DOWNLOAD_ISO** (Lines 65-79)
```groovy
choices: ['Yes', 'No', 'Production']
```
- **Yes**: Download non-production ISO
- **No**: Don't download (use existing)
- **Production**: Download production-ready ISO

**e) ISO_BUILD** (Lines 80-92)
```groovy
name: 'ISO_BUILD'
```
- **Purpose**: Specific ISO version to install
- **Format**: `11793694-2.8.0`
- **Optional**: If empty, uses latest version

**f) Configuration Files** (Lines 140-149)
```groovy
base64File(name: 'rack_data')           // Network configuration
base64File(name: 'devoperator_details') // Developer operator details
base64File(name: 'common_config')       // Pull secrets
base64File(name: 'stage2_config')       // Stage 2 settings
```
- **Type**: File uploads (encoded in base64)
- **Purpose**: Provide configuration data for installation

**g) Storage and Service Options** (Lines 155-241)
```groovy
STORAGE_TYPE: ['Scale', 'ODF', 'Scale+ODF_MCG_Only', ...]
INSTALL_GUARDIAN: ['No', 'Yes']
INSTALL_DISCOVER: ['No', 'Yes']
INSTALL_CAS: ['No', 'Yes']
```
- **Purpose**: Choose which storage systems and services to install

---

### 4. **Global Variables** (Lines 339-445)

```groovy
def rack_array = env.LAPTOPIP_RACKNAME.split("_")
rack_ip = rack_array[0]        // Extract IP: "9.42.56.43"
rack_name = rack_array[1]      // Extract name: "rackname"

env.RACK = rack_ip
env.RACK_NAME = rack_name
```

**Environment Variables Set**:
- `RACK`: The rack IP address
- `RACK_NAME`: The rack identifier
- `ISF_OPERATOR_VERSION`: Version of ISF being installed
- `START`/`STOP`: Stage boundaries (spaces replaced with underscores)

---

### 5. **Helper Functions** (Lines 447-1978)

These are reusable functions that perform specific tasks. Let me explain the most important ones:

#### a) **slack_notify()** (Lines 447-477)
```groovy
def slack_notify(msg, channel, timestamp=null)
```
- **Purpose**: Sends notifications to Slack
- **When used**: Pipeline start, success, failure, stage completion
- **Color coding**: 
  - Green (#00FF00) = Success
  - Red (#FF0000) = Failure
  - Yellow (#FFFF00) = In Progress

#### b) **create_git_issue()** (Lines 490-518)
```groovy
def create_git_issue(name, repository, description, type)
```
- **Purpose**: Automatically creates GitHub issues when failures occur
- **Why**: Tracks problems for the team to fix
- **Contains**: Build URL, rack info, error details

#### c) **create_node_agent()** (Lines 520-549)
```groovy
def create_node_agent(rack_name, rack_ip_addr)
```
- **Purpose**: Creates a Jenkins agent (worker) on the target rack
- **How**: SSH connection to the rack's management laptop
- **Why**: Allows Jenkins to run commands directly on the rack

#### d) **mgen_update()** (Lines 617-639)
```groovy
def mgen_update()
```
- **Purpose**: Updates the manufacturing software (MGEN) on the rack
- **MGEN**: Manufacturing Generation software - the base system software
- **Process**: Downloads and installs latest version if needed

#### e) **validate_build_id()** (Lines 1051-1083)
```groovy
def validate_build_id()
```
- **Purpose**: Checks if the ISF operator build ID is valid
- **Validation**: Ensures format matches expected pattern
- **Sets**: `env.ISF_OPERATOR_VERSION` for use throughout pipeline

#### f) **populate_install_config()** (Lines 977-998)
```groovy
def populate_install_config(rackData)
```
- **Purpose**: Reads rack configuration and sets up network parameters
- **Data**: IP addresses, hostnames, network settings
- **Output**: Creates configuration files for installation

---

### 6. **Metrics and Monitoring** (Lines 1979-2262)

#### Stage Tracking (Lines 1986-2020)
```groovy
def startStageTracking(String stageName)
def stopStageTracking(String stageName, String status)
```
- **Purpose**: Measures how long each stage takes
- **Data collected**: Start time, end time, duration, status
- **Why**: Performance monitoring and optimization

#### Metrics Sending (Lines 2134-2240)
```groovy
def sendMetric(String metricType, String stageName, String status, Integer duration)
```
- **Purpose**: Sends telemetry data to monitoring systems
- **Metrics include**:
  - Stage duration
  - Success/failure rates
  - Build information
  - Resource usage

---

## Pipeline Stages Explained

The actual pipeline execution starts around line 2583 with a `lock` and `node` block:

```groovy
lock(resource: "${env.LAPTOPIP_RACKNAME}", inversePrecedence: false) {
    node(lab_location_agent) {
```

### What This Means:
- **lock**: Ensures only one build runs on a rack at a time (prevents conflicts)
- **node**: Specifies where to run the pipeline (which Jenkins agent)

---

### Stage 1: **Pull ISO** (Lines 2639-2723)

```groovy
stage("Pull ISO") {
    startStageTracking("pull_iso")
    timeout(time: 90, unit: 'MINUTES') {
```

**What happens**:
1. Downloads the ISO image (operating system installer)
2. Validates the ISO build ID
3. Transfers ISO to the rack's management laptop
4. Updates MGEN if needed

**Key Steps**:
```groovy
// Get ISO build ID (if not provided)
isoBuildId = get_build_id("isf-provisioner tag hci", env.OPERATOR_VERSION)

// Setup connection to rack
setup_mgen_agent(rack_name, rack_ip)

// Download ISO
node("mgen-${rack_name}") {
    sh "./pull_iso.sh ${isoBuildId} ${mgenVersion}"
}
```

**Timeout**: 90 minutes (fails if takes longer)

---

### Stage 2: **MGen Cleanup** (Lines 2724-2769)

```groovy
stage("MGen Cleanup") {
    startStageTracking("Cleanup")
    timeout(time: 180, unit: 'MINUTES') {
```

**What happens**:
1. Cleans up any previous installation
2. Resets the rack to factory state
3. Prepares for fresh installation

**Key Command**:
```groovy
sh 'cd fusion-deploy && python3 ./integrated_bvt_fvt/read_test_config.py -t hci_install -s Cleanup'
```
- Runs a Python script that orchestrates the cleanup
- Progress visible at: `http://{rack_ip}:8080`

**Error Handling**:
```groovy
catch(Exception e) {
    log.error("MGen Cleanup failed!")
    caughtException = e
} finally {
    buildArtifactLink = handle_stage_cleanup("cleanup", "http://${rack_ip}:8080")
}
```
- Captures errors
- Archives logs regardless of success/failure
- Creates GitHub issue if fails

**Timeout**: 180 minutes (3 hours)

---

### Stage 3: **Stage 1** (Lines 2770-2820)

```groovy
stage("Stage 1") {
    startStageTracking("stage_1")
    timeout(time: 65, unit: 'MINUTES') {
```

**What happens**:
1. **Network Setup**: Configures network interfaces, VLANs, IP addresses
2. **Base OS Installation**: Installs Red Hat Enterprise Linux (RHEL)
3. **Hardware Configuration**: Sets up storage controllers, network cards

**Key Command**:
```groovy
sh 'cd fusion-deploy && python3 ./integrated_bvt_fvt/read_test_config.py -t hci_install -s Stage_1'
```

**Progress URL**:
```groovy
rack_stage_url = "https://${rack_ip}:${env.PORT}/networksetup"
```
- Web interface showing installation progress
- Different URL for older versions (< 2.9.0): uses `http` instead of `https`

**Timeout**: 65 minutes

---

### Stage 4: **Stage 2** (Lines 2821-2950+)

```groovy
stage("Stage 2") {
    startStageTracking("stage_2")
    timeout(time: 240, unit: 'MINUTES') {
```

**What happens**:
1. **OpenShift Installation**: Installs Red Hat OpenShift Container Platform (OCP)
2. **ISF Operator Deployment**: Installs IBM Spectrum Fusion operator
3. **Cluster Configuration**: Sets up Kubernetes cluster
4. **Certificate Management**: Configures SSL/TLS certificates

**Key Steps**:
```groovy
// Update configuration files
update_config_file()
validate_config_and_certs()

// Enable firewall
enable_firewall()
set_firewall_rule()

// Run Stage 2 installation
sh 'cd fusion-deploy && python3 ./integrated_bvt_fvt/read_test_config.py -t hci_install -s Stage_2'
```

**Special Handling**:
- Offline installation support (if `OFFLINE_INSTALL == true`)
- Custom certificates (if provided)
- Proxy configuration (if needed)
- Different build sources (Dev, Mint, Staging)

**Timeout**: 240 minutes (4 hours) - longest stage

---

### Stage 5: **Storage** (Lines 2950+)

```groovy
stage("Storage") {
    startStageTracking("Storage")
```

**What happens**:
1. **Storage Type Selection**:
   - **Scale**: IBM Spectrum Scale (high-performance file system)
   - **ODF**: OpenShift Data Foundation (Ceph-based storage)
   - **Scale+ODF_MCG_Only**: Hybrid configuration
   - **ODF+GDP_Remote_Mount**: Remote storage mounting

2. **Storage Installation**: Deploys chosen storage solution
3. **Storage Class Configuration**: Sets up Kubernetes storage classes
4. **Validation**: Verifies storage is working

**Key Functions**:
```groovy
set_storage_flags()           // Sets environment variables based on storage type
set_default_storage_class()   // Configures default storage class
```

---

### Stage 6: **Services** (Lines 3000+)

```groovy
stage("Services") {
    startStageTracking("Services")
```

**What happens**:
Installs optional services based on user selection:

1. **Guardian Service**: Data protection and backup
   ```groovy
   if (env.INSTALL_GUARDIAN == 'Yes') {
       // Install Guardian with specified storage class
   }
   ```

2. **Discover Service**: Asset discovery and management
   ```groovy
   if (env.INSTALL_DISCOVER == 'Yes') {
       // Install Discover with specified storage class
   }
   ```

3. **CAS Service**: Copy and Archive Service
   ```groovy
   if (env.INSTALL_CAS == 'Yes') {
       // Install CAS
   }
   ```

4. **Legacy Services** (for ISF < 2.7.0):
   - Service Framework
   - SPP (Backup & Restore Legacy)

---

### Stage 7: **BVT** (Build Verification Test)

```groovy
stage("BVT") {
    startStageTracking("BVT")
```

**What happens**:
1. **Basic Functionality Tests**: Verifies core features work
2. **API Tests**: Tests REST APIs
3. **UI Tests**: Tests web interface
4. **Integration Tests**: Tests component interactions

**Key Function**:
```groovy
call_integrated_bvt_fvt_svt(consoleUrl, ocpKey, storageType, 'bvt')
```

**Pass Criteria**:
- Default: 90% tests must pass
- ODF storage: 75% tests must pass (more lenient)

---

### Stage 8: **FVT** (Functional Verification Test)

```groovy
stage("FVT") {
    startStageTracking("FVT")
```

**What happens**:
1. **Feature Tests**: Tests specific features in detail
2. **Workflow Tests**: Tests complete user workflows
3. **Performance Tests**: Basic performance validation

**Similar to BVT but more comprehensive**

---

### Stage 9: **SVT** (System Verification Test)

```groovy
stage("SVT") {
    startStageTracking("SVT")
```

**What happens**:
1. **System-Level Tests**: Tests entire system behavior
2. **Stress Tests**: Tests under load
3. **Longevity Tests**: Tests stability over time

**Key Function**:
```groovy
execute_svt()
```

---

### Stage 10: **FDF-TESTS** (Fusion Data Foundation Tests)

```groovy
stage("FDF-TESTS") {
    startStageTracking("FDF-TESTS")
```

**What happens**:
1. **OCS-CI Tests**: OpenShift Container Storage tests
2. **Storage Performance**: Tests storage performance
3. **Data Integrity**: Verifies data consistency

**Key Function**:
```groovy
execute_df_ocs_ci_suite()
```

---

## How Everything Works Together

### Execution Flow Diagram

```
User Triggers Pipeline
         ↓
   [Parameters Set]
         ↓
   [Pre-requisites]
    - Validate inputs
    - Clone repositories
    - Setup credentials
         ↓
   [Pull ISO] ← Downloads installation media
         ↓
   [MGen Cleanup] ← Resets rack
         ↓
   [Stage 1] ← Network + Base OS
         ↓
   [Stage 2] ← OpenShift + ISF
         ↓
   [Storage] ← Storage system
         ↓
   [Services] ← Optional services
         ↓
   [BVT] ← Basic tests
         ↓
   [FVT] ← Feature tests
         ↓
   [SVT] ← System tests
         ↓
   [FDF-TESTS] ← Storage tests
         ↓
   [Cleanup & Report]
    - Archive logs
    - Send metrics
    - Notify Slack
    - Create issues (if failed)
```

---

## Key Concepts Explained

### 1. **Credentials Management**
```groovy
withCredentials([
    usernamePassword(credentialsId:'hciops-github-token', ...),
    file(credentialsId: 'hci_rack_automation_devoperator_details', ...)
]) {
    // Code that needs credentials
}
```
- **Purpose**: Securely access passwords, tokens, files
- **How**: Jenkins stores credentials encrypted
- **Scope**: Only available within the `withCredentials` block

### 2. **Node Agents**
```groovy
node("mgen-${rack_name}") {
    // Commands run on the rack's laptop
}
```
- **Purpose**: Execute commands on remote machines
- **How**: Jenkins connects via SSH
- **Why**: Need to run commands directly on the rack hardware

### 3. **Error Handling Pattern**
```groovy
try {
    // Main logic
} catch (Exception e) {
    // Handle error
    caughtException = e
} finally {
    // Cleanup (always runs)
    handle_stage_cleanup()
}
if (caughtException != null) {
    // Create issue, send notifications
    throw caughtException
}
```
- **Purpose**: Graceful error handling
- **Benefits**: Logs are archived even on failure
- **Result**: Team gets notified and issue is tracked

### 4. **Conditional Execution**
```groovy
if (env.START == "Cleanup" && env.STOP != "Test") {
    // Run this stage
} else {
    log.info("Skipping this stage")
}
```
- **Purpose**: Skip stages based on START/STOP parameters
- **Benefit**: Allows partial pipeline runs (e.g., only run tests)

### 5. **Timeouts**
```groovy
timeout(time: 90, unit: 'MINUTES') {
    // Stage logic
}
```
- **Purpose**: Prevent stages from hanging forever
- **Action**: Fails the stage if timeout exceeded
- **Why**: Ensures pipeline doesn't run indefinitely

---

## Important Variables and Their Meanings

| Variable | Purpose | Example Value |
|----------|---------|---------------|
| `rack_ip` | IP address of rack's management laptop | `9.42.56.43` |
| `rack_name` | Identifier for the rack | `rackm03` |
| `ISF_OPERATOR_VERSION` | Version of ISF being installed | `2.8.0-11904029` |
| `OCP_VERSION` | OpenShift version | `4.16` |
| `STORAGE_TYPE` | Type of storage to install | `Scale`, `ODF` |
| `BUILD_SOURCE` | Where to get ISF from | `Dev`, `Mint`, `Staging` |
| `OFFLINE_INSTALL` | Whether this is offline install | `true`, `false` |
| `VERSION_NUM` | Numeric ISF version | `280` (for 2.8.0) |

---

## Common Patterns Used

### 1. **Shell Script Execution**
```groovy
sh '''#!/bin/bash
    echo "Running command"
    cd /some/directory
    ./script.sh
'''
```
- Runs bash commands
- Multi-line scripts use triple quotes `'''`

### 2. **Python Script Execution**
```groovy
sh 'cd fusion-deploy && python3 ./integrated_bvt_fvt/read_test_config.py -t hci_install -s Stage_1'
```
- Runs Python automation scripts
- These scripts do the actual installation work

### 3. **File Operations**
```groovy
if (fileExists('console.log')) {
    def data = readJSON file: 'stats.json'
    writeFile file: 'output.txt', text: 'content'
}
```
- Check file existence
- Read JSON/text files
- Write files

### 4. **String Manipulation**
```groovy
env.START = env.START.replaceAll(' ', '_')  // "Stage 1" → "Stage_1"
def array = env.LAPTOPIP_RACKNAME.split("_") // Split by underscore
clean_name = rack_name.replaceAll("_", "-")  // Replace characters
```

---

## Summary

This Jenkinsfile is a **comprehensive automation pipeline** that:

1. **Takes user inputs** (parameters) for customization
2. **Prepares the environment** (downloads software, validates configs)
3. **Installs the system** in multiple stages (OS → OpenShift → Storage → Services)
4. **Tests the installation** (BVT → FVT → SVT → FDF-TESTS)
5. **Reports results** (Slack, GitHub issues, metrics)
6. **Handles errors gracefully** (logs, notifications, cleanup)

**Key Technologies**:
- **Jenkins**: Automation server
- **Groovy**: Scripting language
- **Python**: Installation scripts
- **Bash**: System commands
- **OpenShift/Kubernetes**: Container platform
- **IBM Spectrum Fusion**: HCI software

**Total Process Time**: 6-10 hours for full installation and testing

---

## Next Steps for Migration to Python

Once you understand this file, we can discuss:
1. Which parts to migrate to Python
2. How to structure a Python-based CI/CD system
3. Alternatives to Jenkins (GitHub Actions, GitLab CI, etc.)
4. How to preserve the same functionality in Python

Would you like me to explain any specific section in more detail?
