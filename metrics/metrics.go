// Package metrics 封装 trpc-go 的 metrics 包：一次 Init 全局生效，
// 业务代码只用简单字符串指标名打点；配套的 PrometheusSink 可把指标
// 汇入 Prometheus 默认注册表，复用服务现有的 /metrics 端口。
package metrics

import (
	"time"

	tmetrics "trpc.group/trpc-go/trpc-go/metrics"
)

// 通用业务指标名（点号分隔，导出为 Prometheus 指标名时自动转为下划线）。
const (
	// NameAnalyticsQueryTotal OJ 分析查询总次数。
	NameAnalyticsQueryTotal = "oj.analytics.query.total"
	// NameAnalyticsQueryErrorsTotal OJ 分析查询失败次数。
	NameAnalyticsQueryErrorsTotal = "oj.analytics.query.errors.total"
)

// Init 注册全局 Sink；未 Init 时所有打点为空操作（与 trpc-go 行为一致）。
func Init(sinks ...tmetrics.Sink) {
	for _, s := range sinks {
		tmetrics.RegisterMetricsSink(s)
	}
}

// ConsoleInit 注册 trpc-go 内置控制台 Sink（按上报打印增量，仅调试用）。
func ConsoleInit() {
	Init(tmetrics.NewConsoleSink())
}

// Incr 计数器加 1。
func Incr(name string) { tmetrics.IncrCounter(name, 1) }

// IncrBy 计数器加 v。
func IncrBy(name string, v float64) { tmetrics.IncrCounter(name, v) }

// Counter 获取（或创建）命名计数器。
func Counter(name string) tmetrics.ICounter { return tmetrics.Counter(name) }

// SetGauge 设置 gauge 为绝对值。
func SetGauge(name string, v float64) { tmetrics.SetGauge(name, v) }

// Gauge 获取（或创建）命名 gauge。
func Gauge(name string) tmetrics.IGauge { return tmetrics.Gauge(name) }

// NewTimer 创建计时器（起点为当前时间）。
func NewTimer(name string) tmetrics.ITimer { return tmetrics.NewTimer(name) }

// RecordTimer 记录一段耗时。
func RecordTimer(name string, d time.Duration) { tmetrics.RecordTimer(name, d) }

// AddSample 向命名直方图添加一个样本（bounds 为桶边界，可重复传同一组）。
func AddSample(name string, bounds tmetrics.BucketBounds, v float64) {
	tmetrics.AddSample(name, bounds, v)
}

// DurationBounds 用时间值构造直方图桶边界。
func DurationBounds(ds ...time.Duration) tmetrics.BucketBounds {
	return tmetrics.NewDurationBounds(ds...)
}

// ValueBounds 用数值构造直方图桶边界。
func ValueBounds(vs ...float64) tmetrics.BucketBounds {
	return tmetrics.NewValueBounds(vs...)
}

// SecondsBounds 把时长桶边界换算成秒。
//
// tmetrics 的 PolicyHistogram 不做单位换算，Prometheus Sink 会把样本值原样落桶，
// 因此 *_seconds 直方图必须用"秒"作样本值与桶边界，不能用 DurationBounds（纳秒）。
func SecondsBounds(ds ...time.Duration) tmetrics.BucketBounds {
	bounds := make(tmetrics.BucketBounds, 0, len(ds))
	for _, d := range ds {
		bounds = append(bounds, d.Seconds())
	}
	return bounds
}

// Report 上报一条（多维）记录。
func Report(rec tmetrics.Record, opts ...tmetrics.Option) error {
	return tmetrics.Report(rec, opts...)
}

// Label 多维上报的一个标签键值对。
// 值域必须是封闭小集合（可枚举）；禁止放入用户 ID、题目 ID 等高基数字段。
type Label struct {
	Name  string
	Value string
}

// IncrWithLabels 计数器加 1，按标签区分序列（多维上报）。
// 例：IncrWithLabels("oj.rcmd.request", Label{"type", "common"})
func IncrWithLabels(name string, labels ...Label) {
	dims := make([]*tmetrics.Dimension, len(labels))
	for i, l := range labels {
		dims[i] = &tmetrics.Dimension{Name: l.Name, Value: l.Value}
	}
	_ = tmetrics.ReportMultiDimensionMetricsX(name, dims,
		[]*tmetrics.Metrics{tmetrics.NewMetrics(name, 1, tmetrics.PolicySUM)})
}
