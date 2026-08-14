// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/autopilot-cost-calculator/calculator"
	"github.com/GoogleCloudPlatform/autopilot-cost-calculator/cluster"
	container "google.golang.org/api/container/v1"
	"gopkg.in/ini.v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

//go:embed config.ini
var defaultConfigBytes []byte

const appVersion = "1.2.0"

func loadConfig(configPath string) (*ini.File, error) {
	if configPath != "" {
		return ini.Load(configPath)
	}

	// Try reading local config.ini if it exists
	if _, err := os.Stat("config.ini"); err == nil {
		return ini.Load("config.ini")
	}

	// Fallback to embedded default configuration
	return ini.Load(defaultConfigBytes)
}

func main() {
	// Flags
	jsonFlag := flag.Bool("json", false, "Output results in JSON format (backward compatible)")
	jsonFileFlag := flag.String("json-file", "", "Path to save JSON output")
	csvFileFlag := flag.String("csv-file", "", "Path to save CSV output")
	markdownFileFlag := flag.String("markdown-file", "", "Path to save Markdown report")
	outputFlag := flag.String("output", "pretty", "Output format: pretty, json, csv, markdown")
	outputShortFlag := flag.String("o", "", "Shorthand for -output")
	monthlyFlag := flag.Bool("monthly", false, "Emphasize monthly pricing across tables and summaries")
	namespaceFlag := flag.String("namespace", "", "Filter workloads by Kubernetes namespace")
	namespaceShortFlag := flag.String("n", "", "Shorthand for -namespace")
	configFileFlag := flag.String("config", "", "Path to custom config.ini file")
	kubeConfigFlag := flag.String("kubeconfig", "", "Path to kubeconfig file")
	contextFlag := flag.String("context", "", "Kubernetes context to use")
	demoFlag := flag.Bool("demo", false, "Run in demo mode with simulated cluster workloads")
	sampleFlag := flag.Bool("sample", false, "Alias for -demo")
	versionFlag := flag.Bool("version", false, "Print application version and exit")
	flag.BoolVar(versionFlag, "v", false, "Shorthand for -version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GKE Autopilot Cost Calculator (v%s)\n\n", appVersion)
		fmt.Fprintf(os.Stderr, "Usage: autopilot-cost-calculator [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("GKE Autopilot Cost Calculator v%s\n", appVersion)
		return
	}

	// Resolve shorthand flags
	outputFormat := strings.ToLower(*outputFlag)
	if *outputShortFlag != "" {
		outputFormat = strings.ToLower(*outputShortFlag)
	}
	if *jsonFlag {
		outputFormat = "json"
	}

	namespaceFilter := *namespaceFlag
	if *namespaceShortFlag != "" {
		namespaceFilter = *namespaceShortFlag
	}

	isDemo := *demoFlag || *sampleFlag

	cfg, err := loadConfig(*configFileFlag)
	if err != nil {
		log.Fatalf("Failed to read configuration: %v", err)
	}

	oneYearDiscount, err := cfg.Section("discounts").Key("oneyear_commit").Float64()
	if err != nil {
		oneYearDiscount = 0.80
	}
	threeYearDiscount, err := cfg.Section("discounts").Key("threeyear_commit").Float64()
	if err != nil {
		threeYearDiscount = 0.55
	}
	clusterFee, err := cfg.Section("fees").Key("cluster_fee").Float64()
	if err != nil {
		clusterFee = calculator.CLUSTER_FEE
	}

	var clusterInfo cluster.ClusterInfo
	var nodes map[string]cluster.Node
	var workloads []cluster.Workload
	var pricingService *calculator.PricingService

	ctx := context.Background()

	if isDemo {
		clusterInfo, nodes = cluster.GetDemoClusterData()
		pricingService = calculator.NewMockService(clusterInfo.Location, cfg)

		for _, node := range nodes {
			workloads = append(workloads, node.Workloads...)
		}
		clusterInfo.NodeCount = len(nodes)
		clusterInfo.WorkloadCount = len(workloads)
	} else {
		// Live Kubernetes cluster inspection
		kubeConfig, kubeConfigPath, err := cluster.GetKubeConfig(*kubeConfigFlag)
		if err != nil {
			log.Fatalf("Error loading Kubernetes configuration: %v\nPlease make sure ~/.kube/config exists or use --demo mode.", err)
		}

		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			log.Fatalf("Error initializing Kubernetes client: %v", err)
		}

		metricsClientset, err := metricsv.NewForConfig(kubeConfig)
		if err != nil {
			log.Fatalf("Error initializing Kubernetes metrics client: %v", err)
		}

		svc, err := container.NewService(ctx)
		if err != nil {
			log.Fatalf("Error initializing GKE client: %v", err)
		}

		currentContext, err := cluster.GetCurrentContext(kubeConfigPath, *contextFlag)
		if err != nil {
			log.Fatalf("Error reading GKE context: %v", err)
		}

		clusterProject := currentContext[1]
		clusterRegion := currentContext[2]
		clusterName := currentContext[3]
		clusterLocation := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", clusterProject, clusterRegion, clusterName)

		clusterObject, err := svc.Projects.Locations.Clusters.Get(clusterLocation).Context(ctx).Do()
		if err != nil {
			log.Fatalf("Error getting GKE cluster information for %s: %v\nEnsure gcloud is authenticated and Cloud Container API is enabled.", clusterName, err)
		}

		if clusterObject.Autopilot != nil && clusterObject.Autopilot.Enabled {
			log.Fatalf("Cluster %q is already in Autopilot mode. This tool calculates migration estimates for GKE Standard clusters.", clusterName)
		}

		nodes, err = cluster.GetClusterNodes(ctx, clientset)
		if err != nil {
			log.Fatalf("Error querying cluster nodes: %v", err)
		}

		pricingSKUs := map[string]string{
			"autopilot": cfg.Section("").Key("autopilot_sku").String(),
			"gce":       cfg.Section("").Key("gce_sku").String(),
		}

		pricingService, err = calculator.NewService(ctx, pricingSKUs, clusterRegion, clientset, metricsClientset, cfg)
		if err != nil {
			log.Fatalf("Error initializing Cloud Billing pricing service: %v\nEnable Cloud Billing API: https://console.cloud.google.com/apis/api/cloudbilling.googleapis.com/metrics", err)
		}

		workloads, err = pricingService.PopulateWorkloads(ctx, nodes, namespaceFilter)
		if err != nil {
			log.Fatalf("Error querying cluster workloads & metrics: %v", err)
		}

		clusterInfo = cluster.ClusterInfo{
			Name:          clusterObject.Name,
			Project:       clusterProject,
			Location:      clusterRegion,
			Status:        clusterObject.Status,
			MasterVersion: clusterObject.CurrentMasterVersion,
			NodeCount:     len(nodes),
			WorkloadCount: len(workloads),
		}
	}

	summary := pricingService.CalculateSummary(nodes, workloads, oneYearDiscount, threeYearDiscount, clusterFee)

	// File export handling
	if *jsonFileFlag != "" {
		f, err := os.Create(*jsonFileFlag)
		if err != nil {
			log.Fatalf("Error creating JSON output file: %v", err)
		}
		defer f.Close()
		if err := ExportJSON(f, nodes, summary, clusterInfo); err != nil {
			log.Fatalf("Error writing JSON file: %v", err)
		}
		DisplayExportNotification("JSON", *jsonFileFlag)
	}

	if *csvFileFlag != "" {
		f, err := os.Create(*csvFileFlag)
		if err != nil {
			log.Fatalf("Error creating CSV output file: %v", err)
		}
		defer f.Close()
		if err := ExportCSV(f, nodes, summary); err != nil {
			log.Fatalf("Error writing CSV file: %v", err)
		}
		DisplayExportNotification("CSV", *csvFileFlag)
	}

	if *markdownFileFlag != "" {
		f, err := os.Create(*markdownFileFlag)
		if err != nil {
			log.Fatalf("Error creating Markdown output file: %v", err)
		}
		defer f.Close()
		if err := ExportMarkdown(f, nodes, summary, clusterInfo); err != nil {
			log.Fatalf("Error writing Markdown file: %v", err)
		}
		DisplayExportNotification("Markdown", *markdownFileFlag)
	}

	// Console output based on format
	switch outputFormat {
	case "json":
		if *jsonFileFlag == "" {
			_ = ExportJSON(os.Stdout, nodes, summary, clusterInfo)
		}
	case "csv":
		if *csvFileFlag == "" {
			_ = ExportCSV(os.Stdout, nodes, summary)
		}
	case "markdown", "md":
		if *markdownFileFlag == "" {
			_ = ExportMarkdown(os.Stdout, nodes, summary, clusterInfo)
		}
	default:
		// Default rich Pretty TUI output
		DisplayHeader(clusterInfo)
		DisplaySummaryCards(summary, *monthlyFlag)
		DisplayWorkloadTable(nodes, *monthlyFlag)
		DisplayNodeTable(nodes, *monthlyFlag)
		DisplayInfoCallout()
	}
}
