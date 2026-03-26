package convert

import (
	"net/url"
	"regexp"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown"
)

var (
	imgTagRegex = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	attrRegex   = regexp.MustCompile(`(?is)([a-zA-Z_:][a-zA-Z0-9_:\-\.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
)

type Converter struct {
	md *htmltomarkdown.Converter
}

func NewConverter() *Converter {
	return &Converter{md: htmltomarkdown.NewConverter("", true, nil)}
}

func (c *Converter) HTMLToMarkdown(html, baseURL string) (string, error) {
	html = imgTagsToLinks(html, baseURL)
	markdown, err := c.md.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(markdown), nil
}

func imgTagsToLinks(html, baseURL string) string {
	base, _ := url.Parse(baseURL)
	return imgTagRegex.ReplaceAllStringFunc(html, func(tag string) string {
		src := parseAttr(tag, "src")
		if src == "" {
			return ""
		}
		resolved := resolveURL(base, src)
		alt := cleanSpace(parseAttr(tag, "alt"))
		if alt == "" {
			alt = "image"
		}
		return `<a href="` + escapeHTMLAttr(resolved) + `">` + escapeHTMLText("image: "+alt) + `</a>`
	})
}

func parseAttr(tag, name string) string {
	name = strings.ToLower(name)
	matches := attrRegex.FindAllStringSubmatch(tag, -1)
	for _, m := range matches {
		if len(m) < 5 {
			continue
		}
		if strings.ToLower(m[1]) != name {
			continue
		}
		for _, v := range m[2:5] {
			if v != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func resolveURL(base *url.URL, target string) string {
	u, err := url.Parse(strings.TrimSpace(target))
	if err != nil {
		return target
	}
	if base == nil {
		return u.String()
	}
	return base.ResolveReference(u).String()
}

func cleanSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func escapeHTMLAttr(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}

func escapeHTMLText(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
