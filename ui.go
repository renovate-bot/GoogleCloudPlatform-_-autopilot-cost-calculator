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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/autopilot-cost-calculator/calculator"
	"github.com/GoogleCloudPlatform/autopilot-cost-calculator/cluster"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	// Color definitions - Modern Google Cloud inspired palette
	colorPrimary   = lipgloss.Color("#4285F4") // GCP Blue
	colorSecondary = lipgloss.Color("#00BCD4") // Autopilot Cyan
	colorSuccess   = lipgloss.Color("#34A853") // Green
	colorWarning   = lipgloss.Color("#FBBC04") // Amber
	colorSpot      = lipgloss.Color("#7C4DFF") // Deep Violet for Spot
	colorMuted     = lipgloss.Color("#9E9E9E") // Muted grey
	colorDarkBg    = lipgloss.Color("#1E293B") // Dark Slate
	colorBorder    = lipgloss.Color("#334155") // Border slate
	colorWhite     = lipgloss.Color("#FFFFFF")
	colorSubtle    = lipgloss.Color("#64748B")

	// Header & Banner styles
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	pillStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	statusRunningPill = pillStyle.
				Foreground(lipgloss.Color("#065F46")).
				Background(lipgloss.Color("#A7F3D0"))

	regionPill = pillStyle.
			Foreground(lipgloss.Color("#1E3A8A")).
			Background(lipgloss.Color("#BFDBFE"))

	versionPill = pillStyle.
			Foreground(lipgloss.Color("#581C87")).
			Background(lipgloss.Color("#E9D5FF"))

	// Card styles
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginRight(1).
			Width(26)

	cardTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	cardValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			MarginTop(0)

	cardSubtextStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	cardSavingsStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSuccess)

	// Section Titles
	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginTop(1).
				MarginBottom(0)

	// Badges for Compute Classes
	badgeGeneralPurpose = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1D4ED8")).
				Background(lipgloss.Color("#DBEAFE")).
				Padding(0, 1).
				Bold(true)

	badgeBalanced = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#047857")).
			Background(lipgloss.Color("#D1FAE5")).
			Padding(0, 1).
			Bold(true)

	badgeScaleout = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6D28D9")).
			Background(lipgloss.Color("#EDE9FE")).
			Padding(0, 1).
			Bold(true)

	badgeScaleoutArm = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0E7490")).
				Background(lipgloss.Color("#CFFAFE")).
				Padding(0, 1).
				Bold(true)

	badgePerformance = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BE185D")).
				Background(lipgloss.Color("#FCE7F3")).
				Padding(0, 1).
				Bold(true)

	badgeAccelerator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#B45309")).
				Background(lipgloss.Color("#FEF3C7")).
				Padding(0, 1).
				Bold(true)

	badgeGPUPod = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C2410C")).
			Background(lipgloss.Color("#FFEDD5")).
			Padding(0, 1).
			Bold(true)

	badgeSpot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#581C87")).
			Background(lipgloss.Color("#E9D5FF")).
			Padding(0, 1).
			Bold(true)

	badgeStandard = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Callout box
	calloutStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)
)

func renderComputeClassBadge(class cluster.ComputeClass) string {
	switch class {
	case cluster.ComputeClassGeneralPurpose:
		return badgeGeneralPurpose.Render("General-purpose")
	case cluster.ComputeClassBalanced:
		return badgeBalanced.Render("Balanced")
	case cluster.ComputeClassScaleout:
		return badgeScaleout.Render("Scale-out")
	case cluster.ComputeClassScaleoutArm:
		return badgeScaleoutArm.Render("Scale-out ARM")
	case cluster.ComputeClassPerformance:
		return badgePerformance.Render("Performance")
	case cluster.ComputeClassAccelerator:
		return badgeAccelerator.Render("Accelerator")
	case cluster.ComputeClassGPUPod:
		return badgeGPUPod.Render("GPU Pod")
	default:
		return class.String()
	}
}

func renderSpotBadge(isSpot bool) string {
	if isSpot {
		return badgeSpot.Render("⚡ SPOT")
	}
	return badgeStandard.Render("Standard")
}

func DisplayHeader(info cluster.ClusterInfo) {
	title := fmt.Sprintf(" ☸️  GKE Autopilot Cost Calculator ")
	banner := bannerStyle.Render(title)

	status := statusRunningPill.Render("● " + strings.ToUpper(info.Status))
	region := regionPill.Render("📍 " + info.Location)
	version := versionPill.Render("v" + info.MasterVersion)

	clusterMeta := fmt.Sprintf("Cluster: %s (%s)   %s  %s  %s",
		lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(info.Name),
		lipgloss.NewStyle().Foreground(colorMuted).Render(info.Project),
		status,
		region,
		version,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		MarginBottom(1).
		Render(banner + "\n" + clusterMeta)

	fmt.Println(box)
}

func DisplaySummaryCards(summary calculator.CostSummary, displayMonthly bool) {
	// Card 1: On Demand
	c1Title := cardTitleStyle.Render("🕒 On-Demand Estimate")
	c1Val := cardValueStyle.Render(fmt.Sprintf("$%.4f / hr", summary.HourlyTotalOnDemand))
	c1Sub := cardSubtextStyle.Render(fmt.Sprintf("$%.2f / month\n(incl. $%.2f cluster fee)", summary.MonthlyTotalOnDemand, summary.MonthlyClusterFee))
	card1 := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, c1Title, c1Val, c1Sub))

	// Card 2: 1-Year Commit CUD
	c2Title := cardTitleStyle.Render("📅 1-Year Commit (20%)")
	c2Val := cardValueStyle.Render(fmt.Sprintf("$%.4f / hr", summary.Hourly1YearCommit))
	c2Sub := cardSavingsStyle.Render(fmt.Sprintf("Save $%.2f / mo\n(-%.1f%%)", summary.Savings1YearMonthly, summary.Savings1YearPercentage))
	card2 := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, c2Title, c2Val, c2Sub))

	// Card 3: 3-Year Commit CUD
	c3Title := cardTitleStyle.Render("🚀 3-Year Commit (45%)")
	c3Val := cardValueStyle.Render(fmt.Sprintf("$%.4f / hr", summary.Hourly3YearCommit))
	c3Sub := cardSavingsStyle.Render(fmt.Sprintf("Save $%.2f / mo\n(-%.1f%%)", summary.Savings3YearMonthly, summary.Savings3YearPercentage))
	card3 := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, c3Title, c3Val, c3Sub))

	// Card 4: Resource Totals
	c4Title := cardTitleStyle.Render("📊 Resource Snapshot")
	vcpus := float64(summary.TotalCpuMcpu) / 1000.0
	memGib := float64(summary.TotalMemoryMib) / 1024.0
	storageGib := float64(summary.TotalStorageMib) / 1024.0
	c4Val := cardValueStyle.Render(fmt.Sprintf("%.2f vCPU | %.1f GiB", vcpus, memGib))
	c4Sub := cardSubtextStyle.Render(fmt.Sprintf("Storage: %.1f GiB\nWorkloads: %d (%d Spot)", storageGib, summary.TotalWorkloads, summary.SpotWorkloadsCount))
	card4 := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, c4Title, c4Val, c4Sub))

	cards := lipgloss.JoinHorizontal(lipgloss.Top, card1, card2, card3, card4)
	fmt.Println(cards)
	fmt.Println()
}

func DisplayWorkloadTable(nodes map[string]cluster.Node, displayMonthly bool) {
	fmt.Println(sectionTitleStyle.Render("📋 Workload Cost Breakdown (GKE Autopilot Mapping)"))

	headers := []string{"Workload", "Namespace", "Class", "Containers", "Spot", "CPU (mCPU)", "Memory", "Storage", "Acc.", "$ / Hour", "$ / Month"}

	var rows [][]string

	for _, node := range nodes {
		for _, w := range node.Workloads {
			memStr := fmt.Sprintf("%d MiB", w.Memory)
			if w.Memory >= 1024 {
				memStr = fmt.Sprintf("%.1f GiB", float64(w.Memory)/1024.0)
			}

			storageStr := fmt.Sprintf("%d MiB", w.Storage)
			if w.Storage >= 1024 {
				storageStr = fmt.Sprintf("%.1f GiB", float64(w.Storage)/1024.0)
			}

			accStr := "-"
			if w.AcceleratorAmount > 0 {
				accModel := w.AcceleratorType
				if accModel == "" {
					accModel = "GPU"
				}
				accStr = fmt.Sprintf("%dx %s", w.AcceleratorAmount, accModel)
			}

			nsStr := w.Namespace
			if nsStr == "" {
				nsStr = "default"
			}

			hourlyCost := fmt.Sprintf("$%.4f", w.Cost)
			monthlyCost := fmt.Sprintf("$%.2f", w.Cost*calculator.HOURS_PER_MONTH)

			rows = append(rows, []string{
				w.Name,
				nsStr,
				renderComputeClassBadge(w.ComputeClass),
				strconv.Itoa(w.Containers),
				renderSpotBadge(node.Spot),
				fmt.Sprintf("%d", w.Cpu),
				memStr,
				storageStr,
				accStr,
				hourlyCost,
				monthlyCost,
			})
		}
	}

	// Sort rows by workload name
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Bold(true).
					Foreground(colorPrimary).
					Padding(0, 1)
			}

			style := lipgloss.NewStyle().Padding(0, 1)

			// Alternate row styling subtle
			if row%2 == 0 {
				style = style.Foreground(lipgloss.Color("#E2E8F0"))
			} else {
				style = style.Foreground(lipgloss.Color("#CBD5E1"))
			}

			// Highlight cost columns
			if col == 9 {
				style = style.Bold(true).Foreground(colorSuccess)
			} else if col == 10 {
				style = style.Bold(true).Foreground(colorSecondary)
			}

			return style
		})

	fmt.Println(t.Render())
	fmt.Println()
}

func DisplayNodeTable(nodes map[string]cluster.Node, displayMonthly bool) {
	fmt.Println(sectionTitleStyle.Render("🖥️  Source Cluster Nodes"))

	headers := []string{"Node Name", "Machine Type", "Region / Zone", "Accelerator", "Spot", "Workloads", "$ / Hour", "$ / Month"}

	var rows [][]string

	// Sort node names
	var nodeNames []string
	for k := range nodes {
		nodeNames = append(nodeNames, k)
	}
	sort.Strings(nodeNames)

	for _, name := range nodeNames {
		node := nodes[name]
		acc := node.Accelerator
		if acc == "" {
			acc = "-"
		}

		hourlyCost := fmt.Sprintf("$%.4f", node.Cost)
		monthlyCost := fmt.Sprintf("$%.2f", node.Cost*calculator.HOURS_PER_MONTH)

		rows = append(rows, []string{
			node.Name,
			node.InstanceType,
			node.Region,
			acc,
			renderSpotBadge(node.Spot),
			strconv.Itoa(len(node.Workloads)),
			hourlyCost,
			monthlyCost,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Bold(true).
					Foreground(colorPrimary).
					Padding(0, 1)
			}
			style := lipgloss.NewStyle().Padding(0, 1)
			if col == 6 {
				style = style.Bold(true).Foreground(colorSuccess)
			} else if col == 7 {
				style = style.Bold(true).Foreground(colorSecondary)
			}
			return style
		})

	fmt.Println(t.Render())
	fmt.Println()
}

func DisplayInfoCallout() {
	text := "💡 Note: Displayed CPU (mCPU), Memory (MiB), and Ephemeral Storage reflect active snapshot requests & utilization.\n   Pricing is estimated based on GKE Autopilot resource request tiers (250mCPU increments, memory ratios, and min limits).\n   GKE Autopilot cluster management fee is $0.10/hour ($73.00/month)."
	fmt.Println(calloutStyle.Render(text))
}

func ExportJSON(w io.Writer, nodes map[string]cluster.Node, summary calculator.CostSummary, clusterInfo cluster.ClusterInfo) error {
	output := map[string]interface{}{
		"cluster":   clusterInfo,
		"summary":   summary,
		"nodes":     nodes,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func ExportCSV(w io.Writer, nodes map[string]cluster.Node, summary calculator.CostSummary) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write headers
	header := []string{"Node", "Workload", "Namespace", "ComputeClass", "Containers", "Spot", "mCPU", "MemoryMiB", "StorageMiB", "AcceleratorType", "AcceleratorAmount", "HourlyCostUSD", "MonthlyCostUSD"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, node := range nodes {
		for _, wl := range node.Workloads {
			row := []string{
				node.Name,
				wl.Name,
				wl.Namespace,
				wl.ComputeClass.String(),
				strconv.Itoa(wl.Containers),
				strconv.FormatBool(node.Spot),
				strconv.FormatInt(wl.Cpu, 10),
				strconv.FormatInt(wl.Memory, 10),
				strconv.FormatInt(wl.Storage, 10),
				wl.AcceleratorType,
				strconv.FormatInt(wl.AcceleratorAmount, 10),
				fmt.Sprintf("%.6f", wl.Cost),
				fmt.Sprintf("%.2f", wl.Cost*calculator.HOURS_PER_MONTH),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}

	// Write summary row
	summaryRow := []string{
		"TOTAL (incl. cluster fee)",
		fmt.Sprintf("%d workloads", summary.TotalWorkloads),
		"-",
		"-",
		"-",
		fmt.Sprintf("%d spot", summary.SpotWorkloadsCount),
		strconv.FormatInt(summary.TotalCpuMcpu, 10),
		strconv.FormatInt(summary.TotalMemoryMib, 10),
		strconv.FormatInt(summary.TotalStorageMib, 10),
		"-",
		strconv.FormatInt(summary.TotalGpus, 10),
		fmt.Sprintf("%.6f", summary.HourlyTotalOnDemand),
		fmt.Sprintf("%.2f", summary.MonthlyTotalOnDemand),
	}
	return csvWriter.Write(summaryRow)
}

func ExportMarkdown(w io.Writer, nodes map[string]cluster.Node, summary calculator.CostSummary, clusterInfo cluster.ClusterInfo) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# GKE Autopilot Cost Calculator Report\n\n"))
	sb.WriteString(fmt.Sprintf("**Cluster:** `%s` | **Project:** `%s` | **Location:** `%s` | **Version:** `v%s`\n\n",
		clusterInfo.Name, clusterInfo.Project, clusterInfo.Location, clusterInfo.MasterVersion))

	sb.WriteString("## Executive Cost Summary\n\n")
	sb.WriteString("| Pricing Tier | Hourly Cost ($/hr) | Monthly Cost ($/mo) | Estimated Savings |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| **On-Demand (Standard)** | `$%.4f` | `$%.2f` | Baseline |\n", summary.HourlyTotalOnDemand, summary.MonthlyTotalOnDemand))
	sb.WriteString(fmt.Sprintf("| **1-Year Commit (CUD -20%%)** | `$%.4f` | `$%.2f` | **Save $%.2f/mo (%.1f%%)** |\n", summary.Hourly1YearCommit, summary.Monthly1YearCommit, summary.Savings1YearMonthly, summary.Savings1YearPercentage))
	sb.WriteString(fmt.Sprintf("| **3-Year Commit (CUD -45%%)** | `$%.4f` | `$%.2f` | **Save $%.2f/mo (%.1f%%)** |\n", summary.Hourly3YearCommit, summary.Monthly3YearCommit, summary.Savings3YearMonthly, summary.Savings3YearPercentage))
	sb.WriteString(fmt.Sprintf("| *Cluster Management Fee* | `$%.4f` | `$%.2f` | Included in totals |\n\n", summary.HourlyClusterFee, summary.MonthlyClusterFee))

	sb.WriteString("## Resource Totals\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Workloads:** %d\n", summary.TotalWorkloads))
	sb.WriteString(fmt.Sprintf("- **Total Nodes:** %d (%d Spot)\n", summary.TotalNodes, summary.SpotWorkloadsCount))
	sb.WriteString(fmt.Sprintf("- **Total CPU:** %.2f vCPUs (%d mCPU)\n", float64(summary.TotalCpuMcpu)/1000.0, summary.TotalCpuMcpu))
	sb.WriteString(fmt.Sprintf("- **Total Memory:** %.2f GiB (%d MiB)\n", float64(summary.TotalMemoryMib)/1024.0, summary.TotalMemoryMib))
	sb.WriteString(fmt.Sprintf("- **Total Storage:** %.2f GiB (%d MiB)\n", float64(summary.TotalStorageMib)/1024.0, summary.TotalStorageMib))
	if summary.TotalGpus > 0 {
		sb.WriteString(fmt.Sprintf("- **Total Accelerators:** %d GPUs\n", summary.TotalGpus))
	}
	sb.WriteString("\n")

	sb.WriteString("## Workload Breakdown\n\n")
	sb.WriteString("| Workload | Namespace | Compute Class | Spot | mCPU | Memory (MiB) | Storage (MiB) | Acc. | Cost ($/hr) | Cost ($/mo) |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, node := range nodes {
		for _, wl := range node.Workloads {
			accStr := "-"
			if wl.AcceleratorAmount > 0 {
				accStr = fmt.Sprintf("%dx %s", wl.AcceleratorAmount, wl.AcceleratorType)
			}
			nsStr := wl.Namespace
			if nsStr == "" {
				nsStr = "default"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %t | %d | %d | %d | %s | `$%.4f` | `$%.2f` |\n",
				wl.Name, nsStr, wl.ComputeClass.String(), node.Spot, wl.Cpu, wl.Memory, wl.Storage, accStr, wl.Cost, wl.Cost*calculator.HOURS_PER_MONTH))
		}
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

func DisplayExportNotification(format string, path string) {
	fmt.Println(lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSuccess).
		Render(fmt.Sprintf("✓ Successfully exported %s report to: %s", strings.ToUpper(format), path)))
}
