package stringhelper

import "regexp"

func StripMarkdownCodeBlocks(input string) string {
	re := regexp.MustCompile("(?s)```.*?```")
	return re.ReplaceAllString(input, "")
}
