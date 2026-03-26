package extract

import "strings"

func Title(html string) string {
	lower := strings.ToLower(html)
	start := strings.Index(lower, "<title")
	if start == -1 {
		return ""
	}
	openEnd := strings.Index(lower[start:], ">")
	if openEnd == -1 {
		return ""
	}
	from := start + openEnd + 1
	end := strings.Index(lower[from:], "</title>")
	if end == -1 {
		return ""
	}
	return cleanSpace(stripTags(html[from : from+end]))
}

func MainContent(html string) string {
	for _, tag := range []string{"main", "article", "body"} {
		if segment, ok := tagSegment(html, tag); ok {
			return segment
		}
	}
	return html
}

func tagSegment(html string, tag string) (string, bool) {
	lower := strings.ToLower(html)
	open := "<" + tag
	close := "</" + tag + ">"
	start := strings.Index(lower, open)
	if start == -1 {
		return "", false
	}
	openEndRel := strings.Index(lower[start:], ">")
	if openEndRel == -1 {
		return "", false
	}
	from := start + openEndRel + 1
	endRel := strings.Index(lower[from:], close)
	if endRel == -1 {
		return "", false
	}
	return html[from : from+endRel], true
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteByte(s[i])
			}
		}
	}
	return b.String()
}

func cleanSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
