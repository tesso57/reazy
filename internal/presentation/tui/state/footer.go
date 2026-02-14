package state

import "strings"

// FooterText returns the footer content for the current session.
func FooterText(session Session, loading bool, aiStatus, statusMessage, helpText string) string {
	lines := make([]string, 0, 3)
	if !loading {
		if msg := strings.TrimSpace(statusMessage); msg != "" {
			lines = append(lines, msg)
		}
		if ai := strings.TrimSpace(aiStatus); ai != "" && (session == ArticleView || session == NewsTopicView || session == DetailView) {
			lines = append(lines, ai)
		}
	}
	if h := strings.TrimSpace(helpText); h != "" {
		lines = append(lines, h)
	}
	return strings.Join(lines, "\n")
}
