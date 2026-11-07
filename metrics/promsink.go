package metrics

import (
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	tmetrics "trpc.group/trpc-go/trpc-go/metrics"
)

// NamePrometheusSink 是 PrometheusSink 的 Sink 名称。
const NamePrometheusSink = "prometheus"

var invalidMetricChars = regexp.MustCompile(`[^a-zA-Z0-9_:]`)

// SanitizeMetricName 将指标名/维度名转换为合法的 Prometheus 名称
// （非法字符替换为下划线；数字开头补下划线；空名归一为 unnamed）。
func SanitizeMetricName(name string) string {
	s := invalidMetricChars.ReplaceAllString(name, "_")
	if s == "" {
		return "unnamed"
	}
	if c := s[0]; c >= '0' && c <= '9' {
		s = "_" + s
	}
	return s
}

// SinkOption 配置 PrometheusSink。
type SinkOption func(*prometheusSinkOptions)

type prometheusSinkOptions struct {
	reg prometheus.Registerer
}

// WithRegistry 指定注册表；默认使用 prometheus 默认注册表，
// 与服务现有 /metrics 端点（DefaultGatherer）共存。
func WithRegistry(r prometheus.Registerer) SinkOption {
	return func(o *prometheusSinkOptions) { o.reg = r }
}

// PrometheusSink 将 trpc-go 指标上报写入 Prometheus 注册表：
//
//	SUM 策略（计数器）  → Counter，累加增量（名字自动补 _total 后缀）
//	SET 策略（gauge）   → Gauge
//	Timer 策略          → Gauge，名字追加 _seconds，值为秒（trpc 原始值为纳秒）
//	Histogram 策略      → Histogram，桶边界来自 HistogramSink.Register
//
// 记录携带维度（多维 Record）时，维度映射为 Prometheus label。
type PrometheusSink struct {
	reg prometheus.Registerer

	mu         sync.Mutex
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	boundsBy   map[string][]float64 // 指标名 → 已注册的直方图桶边界
}

// NewPrometheusSink 创建 Sink；不传选项时注册到 prometheus 默认注册表。
func NewPrometheusSink(opts ...SinkOption) *PrometheusSink {
	o := &prometheusSinkOptions{reg: prometheus.DefaultRegisterer}
	for _, f := range opts {
		f(o)
	}
	return &PrometheusSink{
		reg:        o.reg,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		boundsBy:   make(map[string][]float64),
	}
}

// Name 实现 tmetrics.Sink。
func (s *PrometheusSink) Name() string { return NamePrometheusSink }

// Register 实现 tmetrics.HistogramSink：直方图创建时预置桶边界。
func (s *PrometheusSink) Register(name string, o tmetrics.HistogramOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := make([]float64, len(o.BucketBounds))
	copy(b, o.BucketBounds)
	sort.Float64s(b)
	s.boundsBy[SanitizeMetricName(name)] = b
}

// Report 实现 tmetrics.Sink：把一条记录中的各指标按策略写入注册表。
func (s *PrometheusSink) Report(rec tmetrics.Record, _ ...tmetrics.Option) error {
	dims := rec.GetDimensions()
	var firstErr error
	for _, m := range rec.GetMetrics() {
		if m == nil {
			continue
		}
		name := m.Name()
		if name == "" {
			name = rec.GetName()
		}
		if name == "" {
			continue
		}
		if err := s.record(name, dims, m); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *PrometheusSink) record(name string, dims []*tmetrics.Dimension, m *tmetrics.Metrics) error {
	labels, values := labelPair(dims)
	n := SanitizeMetricName(name)
	switch m.Policy() {
	case tmetrics.PolicySUM:
		c, err := s.counter(ensureTotal(n), labels)
		if err != nil {
			return err
		}
		c.WithLabelValues(values...).Add(m.Value())
	case tmetrics.PolicySET, tmetrics.PolicyAVG, tmetrics.PolicyMAX, tmetrics.PolicyMIN, tmetrics.PolicyMID:
		g, err := s.gauge(n, labels)
		if err != nil {
			return err
		}
		g.WithLabelValues(values...).Set(m.Value())
	case tmetrics.PolicyTimer:
		g, err := s.gauge(n+"_seconds", labels)
		if err != nil {
			return err
		}
		g.WithLabelValues(values...).Set(m.Value() / 1e9)
	case tmetrics.PolicyHistogram:
		h, err := s.histogram(n, labels)
		if err != nil {
			return err
		}
		h.WithLabelValues(values...).Observe(m.Value())
	default:
		// 未定义策略：忽略
	}
	return nil
}

func (s *PrometheusSink) counter(name string, labels []string) (*prometheus.CounterVec, error) {
	key := vecKey(name, labels)
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.counters[key]; ok {
		return v, nil
	}
	v := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, labels)
	if err := s.reg.Register(v); err != nil {
		return nil, err
	}
	s.counters[key] = v
	return v, nil
}

func (s *PrometheusSink) gauge(name string, labels []string) (*prometheus.GaugeVec, error) {
	key := vecKey(name, labels)
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.gauges[key]; ok {
		return v, nil
	}
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, labels)
	if err := s.reg.Register(v); err != nil {
		return nil, err
	}
	s.gauges[key] = v
	return v, nil
}

func (s *PrometheusSink) histogram(name string, labels []string) (*prometheus.HistogramVec, error) {
	key := vecKey(name, labels)
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.histograms[key]; ok {
		return v, nil
	}
	opts := prometheus.HistogramOpts{Name: name}
	if b, ok := s.boundsBy[name]; ok {
		opts.Buckets = b // +Inf 桶由 client_golang 自动补充
	}
	v := prometheus.NewHistogramVec(opts, labels)
	if err := s.reg.Register(v); err != nil {
		return nil, err
	}
	s.histograms[key] = v
	return v, nil
}

// ensureTotal 为计数器名补 Prometheus 惯例的 _total 后缀（client 不会自动补）。
func ensureTotal(name string) string {
	if strings.HasSuffix(name, "_total") {
		return name
	}
	return name + "_total"
}

func vecKey(name string, labels []string) string {
	return name + "|" + strings.Join(labels, "\x00")
}

func labelPair(dims []*tmetrics.Dimension) (labels, values []string) {
	for _, d := range dims {
		if d == nil || d.Name == "" {
			continue
		}
		labels = append(labels, SanitizeMetricName(d.Name))
		values = append(values, d.Value)
	}
	return
}

// Handler 返回带 CORS 的 promhttp 处理器（prometheus 默认注册表），
// 供需要自行挂载 /metrics 的服务使用。
func Handler() http.Handler {
	return withCORS(promhttp.Handler())
}

// JSONHandler 返回 /metrics.json 处理器：把 prometheus 默认注册表的当前值
// 导出为 JSON（带 CORS），供前端跨域直读当前值。
// 格式：{"指标名": 数值, "指标名{l1=v1,l2=v2}": 数值, ...}；
// 直方图导出为 {"_sum":…, "_count":…, "le=<边界>": <累计计数>}。
func JSONHandler() http.Handler { return jsonHandler(prometheus.DefaultGatherer) }

func jsonHandler(g prometheus.Gatherer) http.Handler {
	return withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fams, err := g.Gather()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := map[string]interface{}{}
		for _, fam := range fams {
			fname := fam.GetName()
			for _, m := range fam.GetMetric() {
				key := fname
				if lps := m.GetLabel(); len(lps) > 0 {
					pairs := make([]string, 0, len(lps))
					for _, lp := range lps {
						pairs = append(pairs, lp.GetName()+"="+lp.GetValue())
					}
					sort.Strings(pairs)
					key = fname + "{" + strings.Join(pairs, ",") + "}"
				}
				switch fam.GetType() {
				case dto.MetricType_COUNTER:
					out[key] = m.GetCounter().GetValue()
				case dto.MetricType_GAUGE:
					out[key] = m.GetGauge().GetValue()
				case dto.MetricType_UNTYPED:
					out[key] = m.GetUntyped().GetValue()
				case dto.MetricType_HISTOGRAM:
					h := m.GetHistogram()
					entry := map[string]interface{}{
						"_sum":   h.GetSampleSum(),
						"_count": h.GetSampleCount(),
					}
					for _, b := range h.GetBucket() {
						le := "+Inf"
						if !math.IsInf(b.GetUpperBound(), 1) {
							le = strconv.FormatFloat(b.GetUpperBound(), 'g', -1, 64)
						}
						entry["le="+le] = b.GetCumulativeCount()
					}
					out[key] = entry
				case dto.MetricType_SUMMARY:
					sm := m.GetSummary()
					out[key] = map[string]interface{}{"_sum": sm.GetSampleSum(), "_count": sm.GetSampleCount()}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}))
}

// withCORS 添加跨域头（回显 Origin；无 Origin 时为 *），并直接放行 OPTIONS 预检。
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
