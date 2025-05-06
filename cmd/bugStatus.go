/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/edcdavid/jira-helper/internal/reports"
	"github.com/spf13/cobra"
)

var releaseCutoffDate, FromDate string

// bugStatusCmd represents the bugStatus command
var bugStatusCmd = &cobra.Command{
	Use:   "bugStatus",
	Short: "Creates a markdown bar diagram with bug status",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		reports.GetBugStatusReport(jiraURL, token, releaseCutoffDate, FromDate)
	},
}

func init() {
	rootCmd.AddCommand(bugStatusCmd)
	bugStatusCmd.Flags().StringVarP(&token, "token", "t", "", "The Personal Access Token from Jira")
	bugStatusCmd.Flags().StringVarP(&jiraURL, "url", "u", "https://issues.redhat.com", "The Jira URL")
	bugStatusCmd.Flags().StringVarP(&releaseCutoffDate, "releaseDate", "r", "2025-05-12",
		"The openshift release date (for example, 2025-05-12)")
	bugStatusCmd.Flags().StringVarP(&FromDate, "fromDate", "d", "2023-05-15",
		"The date from which to consider issues created")
}
