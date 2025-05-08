package jirahelper

import (
	"context"
	"os"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/schollz/progressbar/v3"
)

func FetchAllIssues(ctx context.Context, client *jira.Client, jql string, maxResults int) ([]jira.Issue, error) {
	var allIssues []jira.Issue

	// Step 1: Fetch initial page to get total
	options := &jira.SearchOptions{StartAt: 0, MaxResults: 0}
	result, resp, err := client.Issue.Search(ctx, jql, options)
	if err != nil {
		return nil, err
	}
	total := resp.Total
	allIssues = append(allIssues, result...)

	// Step 2: Setup progress bar
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription("Fetching Jira issues..."),
		progressbar.OptionShowCount(),
	)
	_ = bar.Add(len(result))

	// Step 3: Fetch remaining pages
	for startAt := len(result); startAt < total; startAt += maxResults {
		options := &jira.SearchOptions{StartAt: startAt, MaxResults: maxResults}
		page, _, err := client.Issue.Search(ctx, jql, options)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, page...)
		_ = bar.Add(len(page))
		time.Sleep(300 * time.Millisecond) //nolint:mnd
	}

	return allIssues, nil
}
