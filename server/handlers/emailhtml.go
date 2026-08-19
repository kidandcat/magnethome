package handlers

import "strings"

// Viewport + shrink/scroll CSS so inbound mail (often 600–800px, no viewport)
// is readable inside the admin iframe on a phone. Scripts stay disabled via
// the iframe sandbox and CSP on EmailHTML.
const emailViewportMeta = `<meta name="viewport" content="width=device-width, initial-scale=1">`

const emailViewStyle = `<style>
html{margin:0;padding:0;width:100%;max-width:100%;overflow-x:auto;-webkit-overflow-scrolling:touch;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%}
body{margin:0;padding:0;width:auto!important;min-width:0!important;max-width:100%!important;overflow-x:auto;word-wrap:break-word;overflow-wrap:anywhere}
img,video{max-width:100%!important;height:auto!important}
table{max-width:100%!important;min-width:0!important}
table[width]{width:100%!important;max-width:100%!important}
</style>`

// wrapEmailHTML injects a mobile viewport and layout CSS into sender HTML.
// Full documents keep their markup; fragments get a minimal HTML shell.
func wrapEmailHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return wrapEmailFragment("")
	}
	if i := indexFold(raw, "<head"); i >= 0 {
		if gt := strings.Index(raw[i:], ">"); gt >= 0 {
			at := i + gt + 1
			return raw[:at] + emailViewportMeta + emailViewStyle + raw[at:]
		}
	}
	if i := indexFold(raw, "<html"); i >= 0 {
		if gt := strings.Index(raw[i:], ">"); gt >= 0 {
			at := i + gt + 1
			return raw[:at] + "<head>" + emailViewportMeta + emailViewStyle + "</head>" + raw[at:]
		}
	}
	return wrapEmailFragment(raw)
}

func wrapEmailFragment(inner string) string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8">` + emailViewportMeta + emailViewStyle + `</head><body>` + inner + `</body></html>`
}

func indexFold(s, substr string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}
