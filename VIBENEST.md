# go-csp-collector on VibeNest

**NOT_READY** · Source-only pilot pending Free-tier acceptance. Use a site you
control and synthetic browser reports during acceptance.

## Deployment settings

- Select this fork's codex/vibenest-pilot branch and its root Dockerfile.
  Confirm port 8080 and the standard VibeNest HTTPS address before deployment.
- Use /_healthcheck for health checks. Prometheus is available inside the
  container at 127.0.0.1:9090/metrics. Publish only port 8080.
- No database, volume, runtime environment variables or custom start command
  are required. The selected branch and project settings identify the build.
  Its configuration and default filters come from the committed source.
- The pilot serves legacy CSP at /csp and /csp/report-only, Reporting API CSP
  at /reporting-api/csp, and NEL at /nel and /nel/report-only. Use a dedicated
  pilot collector for one site. The public reporting endpoint accepts
  anonymous uploads, so counts alone do not establish who sent them.

## Browser smoke scenario

Serve a test page from a site you control and set the following response
headers. Replace collector.example.test with the pilot's HTTPS hostname.
Use a page containing an inert inline script, such as one that sets a
data attribute on the page. Use only synthetic content.

```http
Reporting-Endpoints: csp-pilot="https://collector.example.test/reporting-api/csp"
Content-Security-Policy-Report-Only: script-src 'none'; report-uri https://collector.example.test/csp/report-only; report-to csp-pilot
```

Open the page in a browser that supports the configured reporting header.
Delivery can be batched. In VibeNest runtime logs, look for
"browser report accepted" with report csp or reporting_api_csp, mode
report_only, and a script directive category. Repeat on the same HTTPS
address after redeploying the same commit to check that the project
configuration and reporting endpoint still work.

## Limits and storage

The whole collector allows two requests per second with a burst of eight,
four concurrent non-health requests, up to 64 accepted public-listener
connections, a 256 KiB POST body and at most 100 entries per batch. Admission
uses a shared budget and ignores forwarded IP headers. Expect HTTP 429 when
the rate or concurrency limit is reached, 413 for oversized bodies, and 422
for batches over 100 entries.

The application writes only fixed categories for accepted reports. It drops
URLs, query metadata, IP addresses, policy text, samples, and raw fields.
The report log has a 16 MiB stdout budget per process, including its budget
exhaustion notice. Accepted reports still update metrics after stdout reaches
the cap. Metrics and the stdout budget reset on process replacement.

No persistent storage is required for this profile. Runtime logs are a
diagnostic tail. Cross-redeploy retention and the host's effective rotation
and size limit are pending confirmation.
