/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/edcdavid/jira-helper/internal/reports"
	"github.com/spf13/cobra"
)

var issueFilter, token, jiraURL string

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Create a report listing red and yellow issues",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		reports.GetMarkdownReport(jiraURL, token, issueFilter)
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVarP(&issueFilter, "issueFilter", "f", "", "The Jira jql filter query")
	reportCmd.Flags().StringVarP(&token, "token", "t", "", "The Personal Access Token from Jira")
	reportCmd.Flags().StringVarP(&jiraURL, "url", "u", "https://issues.redhat.com", "The Jira URL")
	reportCmd.MarkFlagRequired("token")
	reportCmd.MarkFlagRequired("issueFilter")
}
