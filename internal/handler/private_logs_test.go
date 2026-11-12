package handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestPrivateCSPLogIncludesOnlyFixedFields(t *testing.T) {
	logger, recorder := newRecordingLogger(slog.LevelDebug)
	h := NewReportAPICSPHandler(Filter{}, Options{
		PrivateLogs:    true,
		ClientIP:       ClientIPFull,
		MetadataObject: true,
		Logger:         logger,
		Metrics:        newTestMetrics(),
	})

	body := `[{"type":"csp-violation","body":{"documentURL":"https://site.example.test/sample","blockedURL":"inline","sample":"synthetic sample","effectiveDirective":"script-src-elem","disposition":"report"}},{"type":"csp-violation","body":"synthetic malformed item"}]`
	req := httptest.NewRequest(http.MethodPost, "/reporting-api/csp?metadata=synthetic", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("response = %d, want 200", rr.Code)
	}
	want := []logEntry{{
		Level:   slog.LevelInfo,
		Message: "browser report accepted",
		Data: Fields{
			"report":    "reporting_api_csp",
			"mode":      "report_only",
			"directive": "script-src-elem",
		},
	}}
	if got := recorder.Entries(); !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %+v, want %+v", got, want)
	}
}
