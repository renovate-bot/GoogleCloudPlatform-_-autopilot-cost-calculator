![APCostCalculator](assets/logo-apcc.png)
***
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) ![Tests](https://github.com/GoogleCloudPlatform/autopilot-cost-calculator/actions/workflows/test.yaml/badge.svg)

**APCostCalculator** is a tool that provides an accurate estimate of how much your workloads will cost in [GKE Autopilot mode](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview). 

To obtain an estimate, you can connect to your active Kubernetes workloads running on a GKE cluster in Standard mode (or a compatible Kubernetes cluster with `k8s.io/metrics`). APCostCalculator snapshots consumed resources (CPU, Memory, Ephemeral Storage, and GPU Accelerators) and maps them directly to current GKE Autopilot pricing tiers. Resource allocation is calculated based on requests and actual usage rounded to Autopilot billing increments (250 mCPU steps, memory ratio limits, and minimum thresholds).

---

### Features

- **Modern Terminal UI**: Rich CLI output with Google Cloud palette, KPI summary cards, badges for compute classes, and structured breakdown tables powered by Charm Lipgloss.
- **Commitment Discounts**: Automatic calculation of 1-Year (20% CUD) and 3-Year (45% CUD) committed use discounts and projected monthly savings.
- **Spot & Accelerator Support**: Automatic detection and separate pricing for Spot instances and GPU accelerators (NVIDIA T4, L4, A100, H100).
- **Multiple Export Formats**: Export full cost breakdowns to **JSON**, **CSV** (for spreadsheets), or **Markdown** (for documentation/PRs).
- **Zero-Config Standalone Binary**: Embedded default configuration with fallback support; runs anywhere without external file dependencies.
- **Interactive Demo Mode**: Preview and test the tool instantly without GCP credentials using `--demo`.

---

### Sample Output

```
╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│      GKE Autopilot Cost Calculator                                                                            │
│                                                                                                                 │
│ Cluster: prod-ecommerce-cluster (my-gcp-production-project)    ● RUNNING    📍 us-central1    v1.31.2-gke.1200  │
╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
                                                                                                                   
╭──────────────────────────╮ ╭──────────────────────────╮ ╭──────────────────────────╮ ╭──────────────────────────╮ 
│ 🕒 On-Demand Estimate    │ │ 📅 1-Year Commit (20%)   │ │ 🚀 3-Year Commit (45%)   │ │ 📊 Resource Snapshot     │ 
│ $2.8849 / hr             │ │ $2.3416 / hr             │ │ $1.6625 / hr             │ │ 16.50 vCPU | 63.0 GiB    │ 
│ $2105.95 / month         │ │ Save $396.60 / mo        │ │ Save $892.35 / mo        │ │ Storage: 52.0 GiB        │ 
│ (incl. $73.00 cluster    │ │ (-18.8%)                 │ │ (-42.4%)                 │ │ Workloads: 8 (2 Spot)    │ 
│ fee)                     │ ╰──────────────────────────╯ ╰──────────────────────────╯ ╰──────────────────────────╯ 
╰──────────────────────────╯                                                                                        

📋 Workload Cost Breakdown (GKE Autopilot Mapping)
╭───────────────────────────────┬───────────┬───────────────────┬────────────┬───────────┬────────────┬──────────┬──────────┬──────────────┬──────────┬───────────╮
│ Workload                      │ Namespace │ Class             │ Containers │ Spot      │ CPU (mCPU) │ Memory   │ Storage  │ Acc.         │ $ / Hour │ $ / Month │
├───────────────────────────────┼───────────┼───────────────────┼────────────┼───────────┼────────────┼──────────┼──────────┼──────────────┼──────────┼───────────┤
│ analytics-etl-batch-runner-1b │ analytics │  Scale-out ARM    │ 2          │  ⚡ SPOT  │ 2000       │ 8.0 GiB  │ 10.0 GiB │ -            │ $0.0456  │ $33.30    │
│ api-gateway-service-5d6c      │ api       │  General-purpose  │ 1          │ Standard  │ 1000       │ 2.0 GiB  │ 2.0 GiB  │ -            │ $0.0704  │ $51.39    │
│ embeddings-indexer-job-44a    │ ml-models │  GPU Pod          │ 1          │ Standard  │ 2000       │ 8.0 GiB  │ 10.0 GiB │ 1x nvidia-l4 │ $0.8848  │ $645.89   │
│ frontend-web-deployment-78f4  │ frontend  │  General-purpose  │ 2          │ Standard  │ 500        │ 1.0 GiB  │ 1.0 GiB  │ -            │ $0.0352  │ $25.69    │
│ image-resizer-worker-99x2     │ media     │  Scale-out ARM    │ 1          │  ⚡ SPOT  │ 1000       │ 4.0 GiB  │ 2.0 GiB  │ -            │ $0.0228  │ $16.65    │
│ llm-inference-serving-v2      │ ml-models │  GPU Pod          │ 2          │ Standard  │ 4000       │ 16.0 GiB │ 20.0 GiB │ 1x nvidia-l4 │ $1.0916  │ $796.84   │
│ order-processing-engine-9e8a  │ core      │  Performance      │ 3          │ Standard  │ 4000       │ 16.0 GiB │ 5.0 GiB  │ -            │ $0.3928  │ $286.72   │
│ redis-inmemory-cache-0        │ cache     │  Balanced         │ 1          │ Standard  │ 2000       │ 8.0 GiB  │ 2.0 GiB  │ -            │ $0.2417  │ $176.48   │
╰───────────────────────────────┴───────────┴───────────────────┴────────────┴───────────┴────────────┴──────────┴──────────┴──────────────┴──────────┴───────────╯
```

---

### Installation & Building

Prerequisites: [Go](https://go.dev/doc/install) (1.23 or newer).

```bash
# Clone the repository
git clone https://github.com/GoogleCloudPlatform/autopilot-cost-calculator.git
cd autopilot-cost-calculator

# Build binary
go build -o apcc .
```

---

### Usage

Before running the tool against a live cluster, make sure you have the [gcloud CLI](https://cloud.google.com/sdk/docs/install) installed and configured.

1. **Enable Cloud Billing API on your project:**
   https://console.cloud.google.com/apis/api/cloudbilling.googleapis.com/metrics?project=PROJECT_NAME

2. **Authenticate with GCP:**
   ```bash
   gcloud auth application-default login
   ```

3. **Get credentials for your target GKE cluster:**
   ```bash
   gcloud container clusters get-credentials CLUSTER_NAME --zone ZONE --project PROJECT_NAME
   ```

4. **Run the calculator:**
   ```bash
   # Run against active cluster
   ./apcc

   # Filter by namespace
   ./apcc -n default

   # Run instant demo mode (no credentials required)
   ./apcc --demo

   # Export to JSON, CSV, or Markdown
   ./apcc --demo --output=json
   ./apcc --demo --csv-file=costs.csv
   ./apcc --demo --markdown-file=report.md
   ```

---

### Command-Line Flags

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--output` | `-o` | Output format (`pretty`, `json`, `csv`, `markdown`) | `pretty` |
| `--demo` / `--sample` | | Run with simulated sample cluster data | `false` |
| `--namespace` | `-n` | Filter workloads by Kubernetes namespace | `""` (all) |
| `--monthly` | `-m` | Emphasize monthly cost projections | `false` |
| `--json-file` | | Save JSON report to specified file path | `""` |
| `--csv-file` | | Save CSV export to specified file path | `""` |
| `--markdown-file` | | Save Markdown report to specified file path | `""` |
| `--config` | `-c` | Custom `config.ini` file path | Embedded default |
| `--kubeconfig` | | Explicit path to `kubeconfig` file | `~/.kube/config` |
| `--context` | | Target Kubernetes context name | Active context |
| `--version` | `-v` | Display version information | |

---

### Pricing Reference

For more information about GKE Autopilot pricing and compute classes:
- [GKE Autopilot Pricing](https://cloud.google.com/kubernetes-engine/pricing#autopilot_pricing)
- [GKE Autopilot Resource Requests & Limits](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests)
