/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/edcdavid/jira-helper/internal/reports"
	"github.com/spf13/cobra"
)

var issueFilter, token, jiraURL, release, customerFacing, ollamaModel string
var showOriginalStatus bool

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Create a report listing red and yellow issues",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		reports.GetMarkdownReport(jiraURL, token, issueFilter, release, customerFacing, ollamaModel, showOriginalStatus)
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVarP(&token, "token", "t", "", "The Personal Access Token from Jira")
	reportCmd.Flags().StringVarP(&jiraURL, "url", "u", "https://issues.redhat.com", "The Jira URL")
	reportCmd.Flags().StringVarP(&issueFilter, "issueFilter", "f", "", "The Jira jql filter query")
	reportCmd.Flags().StringVarP(&release, "release", "r", "4.20", "The openshift release (for example, 4.20)")
	reportCmd.Flags().StringVarP(&customerFacing, "customerFacing", "c", "both",
		"yes for customer facing, not for not customer facing, and both for both")
	reportCmd.Flags().StringVarP(&ollamaModel, "ollamaModel", "m", "",
		"Use specified model in Ollama to clean suummary status")
	reportCmd.Flags().BoolVarP(&showOriginalStatus, "originalStatus", "o", false, "Add the original status summary in code blocks")
}
