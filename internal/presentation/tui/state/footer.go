package state

import "strings"

// FooterText returns the footer content for the current session.
func FooterText(session Session, loading bool, aiStatus, helpText string) string {
	status := strings.TrimSpace(aiStatus)
	if !loading && status != "" && (session == ArticleView || session == DetailView) {
		if helpText == "" {
			return status
		}
		return status + "\n" + helpText
	}
	return helpText
}
