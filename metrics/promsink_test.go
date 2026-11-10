package metrics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	tmetrics "trpc.group/trpc-go/trpc-go/metrics"
)

// newTestSink 创建独立注册表上的 Sink 并注册为 trpc-go 全局 Sink。
func newTestSink(t *testing.T) (*PrometheusSink, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	s := NewPrometheusSink(WithRegistry(reg))
	Init(s)
	return s, reg
}

func regText(t *testing.T, reg *prometheus.Registry) string {
	t.Helper()
	fams, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var buf bytes.Buffer
	enc := expfmt.NewEncoder(&buf, expfmt.FmtText)
	for _, f := range fams {
		if err := enc.Encode(f); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	return buf.String()
}

func TestCounterIncr(t *testing.T) {
	_, reg := newTestSink(t)
	Incr("oj.test.counter")
	Incr("oj.test.counter")
	IncrBy("oj.test.counter", 1.5)
	text := regText(t, reg)
	if !strings.Contains(text, "oj_test_counter_total 3.5") {
		t.Fatalf("counter 值不符: \n%s", text)
	}
}

func TestGaugeSet(t *testing.T) {
	_, reg := newTestSink(t)
	SetGauge("oj.test.gauge", 2.25)
	text := regText(t, reg)
	if !strings.Contains(text, "oj_test_gauge 2.25") {
		t.Fatalf("gauge 值不符: \n%s", text)
	}
}

func TestTimerSeconds(t *testing.T) {
	_, reg := newTestSink(t)
	RecordTimer("oj.test.timer", 1500*time.Millisecond)
	text := regText(t, reg)
	if !strings.Contains(text, "oj_test_timer_seconds 1.5") {
		t.Fatalf("timer 值不符: \n%s", text)
	}
}

func TestHistogram(t *testing.T) {
	_, reg := newTestSink(t)
	AddSample("oj.test.hist", DurationBounds(time.Second, 2*time.Second), 1.5e9)
	AddSample("oj.test.hist", DurationBounds(time.Second, 2*time.Second), 0.5e9)
	text := regText(t, reg)
	for _, want := range []string{
		`oj_test_hist_bucket{le="1e+09"} 1`,
		`oj_test_hist_bucket{le="2e+09"} 2`,
		`oj_test_hist_bucket{le="+Inf"} 2`,
		"oj_test_hist_count 2",
		"oj_test_hist_sum 2e+09",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("直方图缺少 %q: \n%s", want, text)
		}
	}
}

func TestMultiDimensionRecord(t *testing.T) {
	_, reg := newTestSink(t)
	rec := tmetrics.NewMultiDimensionMetricsX("dim.record",
		[]*tmetrics.Dimension{{Name: "region", Value: "sh"}},
		[]*tmetrics.Metrics{tmetrics.NewMetrics("req.num", 3, tmetrics.PolicySUM)},
	)
	if err := tmetrics.Report(rec); err != nil {
		t.Fatalf("report: %v", err)
	}
	text := regText(t, reg)
	if !strings.Contains(text, `req_num_total{region="sh"} 3`) {
		t.Fatalf("多维记录不符: \n%s", text)
	}
}

func TestJSONHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	Init(NewPrometheusSink(WithRegistry(reg)))
	Incr("oj.test.json")
	h := jsonHandler(reg)

	req := httptest.NewRequest(http.MethodGet, "/metrics.json", nil)
	req.Header.Set("Origin", "http://localhost:5174")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5174" {
		t.Fatalf("CORS 头不符: %q", got)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if v, ok := out["oj_test_json_total"].(float64); !ok || v != 1 {
		t.Fatalf("oj_test_json_total = %v (全部: %v)", out["oj_test_json_total"], out)
	}
}

func TestCORSOptions(t *testing.T) {
	reg := prometheus.NewRegistry()
	h := jsonHandler(reg)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/metrics.json", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 应返回 204，实际 %d", rec.Code)
	}
}

func TestSanitizeMetricName(t *testing.T) {
	cases := map[string]string{
		"oj.analytics.query.total": "oj_analytics_query_total",
		"9lives":                   "_9lives",
		"a-b.c:d":                  "a_b_c:d",
		"":                         "unnamed",
	}
	for in, want := range cases {
		if got := SanitizeMetricName(in); got != want {
			t.Errorf("SanitizeMetricName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMultiDimensionLabelOrder 校验同一指标名在不同维度顺序下只产生一个序列：
// 维度顺序不同不应导致重复注册，而应归一到同一组 label 与同一个序列。
func TestMultiDimensionLabelOrder(t *testing.T) {
	_, reg := newTestSink(t)

	report := func(dims []*tmetrics.Dimension, v float64) error {
		rec := tmetrics.NewMultiDimensionMetricsX("dim.order",
			dims,
			[]*tmetrics.Metrics{tmetrics.NewMetrics("req.order", v, tmetrics.PolicySUM)},
		)
		return tmetrics.Report(rec)
	}
	if err := report([]*tmetrics.Dimension{
		{Name: "service", Value: "s1"},
		{Name: "code", Value: "ok"},
	}, 1); err != nil {
		t.Fatalf("首次上报失败: %v", err)
	}
	if err := report([]*tmetrics.Dimension{
		{Name: "code", Value: "ok"},
		{Name: "service", Value: "s1"},
	}, 2); err != nil {
		t.Fatalf("维度顺序不同时的上报失败: %v", err)
	}

	fams, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, f := range fams {
		if f.GetName() != "req_order_total" {
			continue
		}
		if len(f.GetMetric()) != 1 {
			t.Fatalf("应只有 1 个序列，实际 %d: %v", len(f.GetMetric()), f.GetMetric())
		}
		if got := f.GetMetric()[0].GetCounter().GetValue(); got != 3 {
			t.Fatalf("累计值应为 3，实际 %v", got)
		}
		return
	}
	t.Fatalf("未找到指标 req_order_total: %s", regText(t, reg))
}

func TestIncrWithLabels(t *testing.T) {
	_, reg := newTestSink(t)
	IncrWithLabels("req.label", Label{"type", "common"})
	IncrWithLabels("req.label", Label{"type", "common"})
	IncrWithLabels("req.label", Label{"type", "contest"})
	text := regText(t, reg)
	for _, want := range []string{
		`req_label_total{type="common"} 2`,
		`req_label_total{type="contest"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("带标签计数缺少 %q: \n%s", want, text)
		}
	}
}
