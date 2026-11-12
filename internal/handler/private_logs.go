package handler

import "log/slog"

// Only fixed category strings reach stdout. Report fields are never copied
// through, even when the browser has already truncated them.
func privateReportAttrs(kind string, result ProcessResult) []slog.Attr {
	attrs := []slog.Attr{slog.String("report", kind), slog.String("mode", result.Mode)}
	switch kind {
	case "csp", "reporting_api_csp":
		directive := "other"
		for _, attr := range result.Attrs {
			if attr.Key == "effective_directive" || (directive == "other" && attr.Key == "violated_directive") {
				directive = privateCSPDirective(attr.Value.String())
			}
		}
		attrs = append(attrs, slog.String("directive", directive))
	case "nel":
		phase := "other"
		for _, attr := range result.Attrs {
			if attr.Key == "phase" {
				switch attr.Value.String() {
				case "dns":
					phase = "dns"
				case "connection":
					phase = "connection"
				case "application":
					phase = "application"
				}
				break
			}
		}
		attrs = append(attrs, slog.String("phase", phase))
	}
	return attrs
}

func privateCSPDirective(value string) string {
	switch value {
	case "default-src", "script-src", "script-src-elem", "script-src-attr",
		"style-src", "style-src-elem", "style-src-attr", "img-src", "font-src",
		"connect-src", "media-src", "object-src", "frame-src", "frame-ancestors",
		"child-src", "worker-src", "manifest-src", "base-uri", "form-action",
		"require-trusted-types-for", "trusted-types":
		return value
	default:
		return "other"
	}
}
