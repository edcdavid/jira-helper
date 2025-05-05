# jira-helper

This tool pull issues from Jira using the rest API and generates a Markdown output with the set of issues that have a red and yellow status. 2 diagram summarize the color status and the workflow status for all issues.

Command syntax is below:
```
build/jira-helper report -h
Create a report listing red and yellow issues

Usage:
  jira-helper report [flags]

Flags:
  -c, --customerFacing string   yes for customer facing, not for not customer facing, and both for both (default "both")
  -h, --help                    help for report
  -f, --issueFilter string      The Jira jql filter query
  -r, --release string          The openshift release (for example, 4.20) (default "4.20")
  -t, --token string            The Personal Access Token from Jira
  -u, --url string              The Jira URL (default "https://issues.redhat.com")
```

Example: to pass the issue filter, escape " character with \". For instance, `project = "OpenShift` becomes `project = \"OpenShift `
```
build/jira-helper report  --token zGvHYRPqABmDsXZfEuLJtbNwgCVehYkqpxoWaUcnKdIqM  --release 4.20 -c yes > test.md
```