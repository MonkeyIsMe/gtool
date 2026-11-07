package metrics

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	tmetrics "trpc.group/trpc-go/trpc-go/metrics"
)

// recorderSink 捕获所有上报记录，便于断言指标名、维度与取值。
type recorderSink struct {
	mu      sync.Mutex
	records []tmetrics.Record
	bounds  map[string][]float64
}

func (s *recorderSink) Name() string { return "test-recorder" }

func (s *recorderSink) Report(rec tmetrics.Record, _ ...tmetrics.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, rec)
	return nil
}

func (s *recorderSink) Register(name string, o tmetrics.HistogramOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bounds == nil {
		s.bounds = make(map[string][]float64)
	}
	s.bounds[SanitizeMetricName(name)] = append([]float64(nil), o.BucketBounds...)
}

func (s *recorderSink) snapshot() []tmetrics.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tmetrics.Record(nil), s.records...)
}

func (s *recorderSink) boundsOf(name string) []float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float64(nil), s.bounds[SanitizeMetricName(name)]...)
}

func (s *recorderSink) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = nil
}

var (
	globalRecorder = &recorderSink{}
	recorderOnce   sync.Once
)

// useRecorder 注册全局捕获 Sink（仅一次）并清空历史记录。
func useRecorder() *recorderSink {
	recorderOnce.Do(func() { Init(globalRecorder) })
	globalRecorder.reset()
	return globalRecorder
}

func hasMetric(rec *tmetrics.Record, name string) bool {
	for _, m := range rec.GetMetrics() {
		if m != nil && m.Name() == name {
			return true
		}
	}
	return false
}

// lastNamed 返回最后一条包含指定指标的记录（按上报顺序）。
func lastNamed(recs []tmetrics.Record, name string) *tmetrics.Record {
	for i := len(recs) - 1; i >= 0; i-- {
		if hasMetric(&recs[i], name) {
			return &recs[i]
		}
	}
	return nil
}

func dimsOf(rec *tmetrics.Record) map[string]string {
	out := make(map[string]string)
	if rec == nil {
		return out
	}
	for _, d := range rec.GetDimensions() {
		if d != nil {
			out[d.Name] = d.Value
		}
	}
	return out
}

func metricOf(rec *tmetrics.Record, name string) *tmetrics.Metrics {
	if rec == nil {
		return nil
	}
	for _, m := range rec.GetMetrics() {
		if m != nil && m.Name() == name {
			return m
		}
	}
	return nil
}

func inflightValues(recs []tmetrics.Record) []float64 {
	var out []float64
	for i := range recs {
		if m := metricOf(&recs[i], NameRPCInflight); m != nil {
			out = append(out, m.Value())
		}
	}
	return out
}

// rpcCtx 构造带 trpc 消息的 context（RPC 名/被调服务对被调方法）。
func rpcCtx(rpcName, calleeService, calleeMethod string) context.Context {
	ctx, msg := codec.WithNewMessage(context.Background())
	if rpcName != "" {
		msg.WithServerRPCName(rpcName)
		msg.WithClientRPCName(rpcName)
	}
	if calleeService != "" {
		msg.WithCalleeServiceName(calleeService)
	}
	if calleeMethod != "" {
		msg.WithCalleeMethod(calleeMethod)
	}
	return ctx
}

func checkDims(t *testing.T, rec *tmetrics.Record, want map[string]string) {
	t.Helper()
	if rec == nil {
		t.Fatalf("没有捕获到记录")
	}
	got := dimsOf(rec)
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("维度 %s = %q，期望 %q（实际全部：%v）", k, got[k], v, got)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("维度数量 %d，期望 %d（实际全部：%v）", len(got), len(want), got)
	}
}

// TestServerFilterRecordsSuccess 服务端成功调用应记录请求数、耗时（秒）与并发数。
func TestServerFilterRecordsSuccess(t *testing.T) {
	rec := useRecorder()
	ctx := rpcCtx("/trpc.oj.agent.AgentServer/Socratic", "", "")
	f := ServerFilter()

	rsp, err := f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(20 * time.Millisecond)
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("成功调用不应返回错误: %v", err)
	}
	if rsp != "ok" {
		t.Fatalf("响应未透传: %v", rsp)
	}

	recs := rec.snapshot()
	checkDims(t, lastNamed(recs, NameRPCRequests), map[string]string{
		"side": "server", "service": "trpc.oj.agent.AgentServer", "method": "Socratic", "code": "ok",
	})
	if m := metricOf(lastNamed(recs, NameRPCRequests), NameRPCRequests); m == nil || m.Value() != 1 || m.Policy() != tmetrics.PolicySUM {
		t.Fatalf("请求计数应为 SUM=1，实际 %+v", m)
	}

	durRec := lastNamed(recs, NameRPCDuration)
	checkDims(t, durRec, map[string]string{
		"side": "server", "service": "trpc.oj.agent.AgentServer", "method": "Socratic",
	})
	if m := metricOf(durRec, NameRPCDuration); m == nil || m.Policy() != tmetrics.PolicyHistogram {
		t.Fatalf("耗时应以直方图上报，实际 %+v", m)
	} else if m.Value() < 0.015 || m.Value() > 1 {
		t.Fatalf("耗时样本应为秒（20ms ≈ 0.02），实际 %v", m.Value())
	}

	checkDims(t, lastNamed(recs, NameRPCInflight), map[string]string{
		"side": "server", "service": "trpc.oj.agent.AgentServer",
	})
}

// TestServerFilterInflight 并发数应在调用中为 1、调用后回到 0。
func TestServerFilterInflight(t *testing.T) {
	rec := useRecorder()
	ctx := rpcCtx("/trpc.oj.agent.AgentServer/Socratic", "", "")
	f := ServerFilter()

	var during []float64
	if _, err := f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		during = inflightValues(rec.snapshot())
		return nil, nil
	}); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	if len(during) == 0 || during[len(during)-1] != 1 {
		t.Fatalf("调用中并发数应为 1，实际 %v", during)
	}
	after := inflightValues(rec.snapshot())
	if len(after) == 0 || after[len(after)-1] != 0 {
		t.Fatalf("调用后并发数应为 0，实际 %v", after)
	}
	if m := metricOf(lastNamed(rec.snapshot(), NameRPCInflight), NameRPCInflight); m == nil || m.Policy() != tmetrics.PolicySET {
		t.Fatalf("并发数应为 SET（gauge），实际 %+v", m)
	}
}

// TestServerFilterErrorCode 错误码 label 使用 TRPC_ 前缀小写后的短名。
func TestServerFilterErrorCode(t *testing.T) {
	cases := []struct {
		desc string
		err  error
		want string
	}{
		{"系统错误", errs.New(errs.RetServerSystemErr, "boom"), "server_system_err"},
		{"方法不存在", errs.New(errs.RetServerNoFunc, "nofunc"), "server_nofunc_err"},
		{"服务端超时", errs.New(errs.RetServerTimeout, "timeout"), "server_timeout_err"},
		{"连接失败", errs.New(errs.RetClientConnectFail, "dial fail"), "client_connect_err"},
		{"普通错误", errors.New("plain"), "invoke_unknown_err"},
	}
	for _, c := range cases {
		rec := useRecorder()
		ctx := rpcCtx("/trpc.oj.agent.AgentServer/Socratic", "", "")
		f := ServerFilter()

		got, err := f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, c.err
		})
		if got != nil {
			t.Fatalf("%s：失败响应应为 nil", c.desc)
		}
		if err != c.err {
			t.Fatalf("%s：错误不应被改写，实际 %v", c.desc, err)
		}
		checkDims(t, lastNamed(rec.snapshot(), NameRPCRequests), map[string]string{
			"side": "server", "service": "trpc.oj.agent.AgentServer", "method": "Socratic", "code": c.want,
		})
	}
}

// TestServerFilterPanic panic 应记为 code=panic 并继续向上抛出。
func TestServerFilterPanic(t *testing.T) {
	rec := useRecorder()
	ctx := rpcCtx("/trpc.oj.agent.AgentServer/Socratic", "", "")
	f := ServerFilter()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("panic 必须继续向上抛出，供 recovery filter 处理")
			}
		}()
		_, _ = f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("boom")
		})
	}()

	recs := rec.snapshot()
	checkDims(t, lastNamed(recs, NameRPCRequests), map[string]string{
		"side": "server", "service": "trpc.oj.agent.AgentServer", "method": "Socratic", "code": "panic",
	})
	if v := inflightValues(recs); len(v) == 0 || v[len(v)-1] != 0 {
		t.Fatalf("panic 后并发数应回到 0，实际 %v", v)
	}
}

// TestClientFilterRecordsDownstream 客户端调用记录 side=client 与被调服务的耗时。
func TestClientFilterRecordsDownstream(t *testing.T) {
	rec := useRecorder()
	ctx := rpcCtx("/trpc.oj.recall.RecallServer/Recall", "trpc.oj.recall.RecallServer", "Recall")
	f := ClientFilter()

	if err := f(ctx, nil, nil, func(ctx context.Context, req, rsp interface{}) error { return nil }); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	recs := rec.snapshot()
	checkDims(t, lastNamed(recs, NameRPCRequests), map[string]string{
		"side": "client", "service": "trpc.oj.recall.RecallServer", "method": "Recall", "code": "ok",
	})
	checkDims(t, lastNamed(recs, NameRPCDuration), map[string]string{
		"side": "client", "service": "trpc.oj.recall.RecallServer", "method": "Recall",
	})
	checkDims(t, lastNamed(recs, NameRPCInflight), map[string]string{
		"side": "client", "service": "trpc.oj.recall.RecallServer",
	})
}

// TestFilterCalleeFallback RPC 名为空时回退到被调服务名+方法。
func TestFilterCalleeFallback(t *testing.T) {
	rec := useRecorder()
	ctx := rpcCtx("", "trpc.oj.dao.MySQL", "Query")
	f := ServerFilter()

	if _, err := f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	checkDims(t, lastNamed(rec.snapshot(), NameRPCRequests), map[string]string{
		"side": "server", "service": "trpc.oj.dao.MySQL", "method": "Query", "code": "ok",
	})
}

// TestFilterNoMessage ctx 无 trpc 消息时不应 panic，且退化为 unknown。
func TestFilterNoMessage(t *testing.T) {
	rec := useRecorder()
	f := ServerFilter()
	if _, err := f(context.Background(), nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	checkDims(t, lastNamed(rec.snapshot(), NameRPCRequests), map[string]string{
		"side": "server", "service": "unknown", "method": "unknown", "code": "ok",
	})
}

func TestParseRPCName(t *testing.T) {
	cases := []struct {
		in      string
		service string
		method  string
	}{
		{"/trpc.oj.agent.AgentServer/Socratic", "trpc.oj.agent.AgentServer", "Socratic"},
		{"trpc.oj.agent.AgentServer/Socratic", "trpc.oj.agent.AgentServer", "Socratic"},
		{"/trpc.oj.a.S/M/extra", "trpc.oj.a.S", "M/extra"},
		{"//M", "unknown", "M"},
		{"noslash", "noslash", "unknown"},
		{"/", "unknown", "unknown"},
		{"", "unknown", "unknown"},
	}
	for _, c := range cases {
		service, method := parseRPCName(c.in)
		if service != c.service || method != c.method {
			t.Errorf("parseRPCName(%q) = (%q, %q)，期望 (%q, %q)", c.in, service, method, c.service, c.method)
		}
	}
}

// TestSecondsBounds 时长桶边界必须换算为秒（PolicyHistogram 不做单位换算）。
func TestSecondsBounds(t *testing.T) {
	got := []float64(SecondsBounds(5*time.Millisecond, 250*time.Millisecond, time.Second))
	want := []float64{0.005, 0.25, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SecondsBounds = %v，期望 %v", got, want)
	}
}

// TestDefaultRPCDurationBounds 默认毫秒级分桶（以秒为单位）。
func TestDefaultRPCDurationBounds(t *testing.T) {
	rec := useRecorder()
	_ = ServerFilter()
	want := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5}
	if got := rec.boundsOf(NameRPCDuration); !reflect.DeepEqual(got, want) {
		t.Fatalf("默认耗时桶 = %v，期望 %v", got, want)
	}
}

// TestRPCOptions 自定义前缀与桶边界生效。
func TestRPCOptions(t *testing.T) {
	rec := useRecorder()
	const prefix = "oj.rpc.customtest"
	f := ServerFilter(WithRPCPrefix(prefix), WithRPCDurationBounds(ValueBounds(0.1, 0.5)))

	if got, want := rec.boundsOf(prefix+".duration.seconds"), []float64{0.1, 0.5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("自定义桶 = %v，期望 %v", got, want)
	}
	ctx := rpcCtx("/trpc.oj.a.S/M", "", "")
	if _, err := f(ctx, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	recs := rec.snapshot()
	checkDims(t, lastNamed(recs, prefix+".requests"), map[string]string{
		"side": "server", "service": "trpc.oj.a.S", "method": "M", "code": "ok",
	})
	if lastNamed(recs, prefix+".duration.seconds") == nil {
		t.Fatalf("缺少 %s 记录", prefix+".duration.seconds")
	}
}

// TestRegisterRPCFilter 注册后可从 filter 注册表按名字取到。
func TestRegisterRPCFilter(t *testing.T) {
	const name = "rpc.metrics.test"
	RegisterRPCFilter(name, WithRPCPrefix("oj.rpc.regtest"))
	if filter.GetServer(name) == nil {
		t.Fatalf("server filter 未注册")
	}
	if filter.GetClient(name) == nil {
		t.Fatalf("client filter 未注册")
	}
}
