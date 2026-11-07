package metrics

import (
	"context"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	tmetrics "trpc.group/trpc-go/trpc-go/metrics"
)

const (
	// NameRPCRequests RPC 请求数（counter，维度 side/service/method/code）。
	NameRPCRequests = "oj.rpc.requests"
	// NameRPCDuration RPC 耗时（histogram，单位秒，维度 side/service/method）。
	NameRPCDuration = "oj.rpc.duration.seconds"
	// NameRPCInflight RPC 并发数（gauge，维度 side/service）。
	NameRPCInflight = "oj.rpc.inflight.requests"
	// DefaultRPCPrefix 指标名前缀。
	DefaultRPCPrefix = "oj.rpc"
)

const (
	rpcSideServer = "server"
	rpcSideClient = "client"
	rpcCodeOK     = "ok"
	rpcCodePanic  = "panic"
	rpcUnknown    = "unknown"
)

// DefaultRPCDurationBounds 默认耗时桶：5ms~2.5s，单位为秒。
func DefaultRPCDurationBounds() tmetrics.BucketBounds {
	return SecondsBounds(5*time.Millisecond, 10*time.Millisecond, 25*time.Millisecond,
		50*time.Millisecond, 100*time.Millisecond, 250*time.Millisecond,
		500*time.Millisecond, time.Second, 2500*time.Millisecond)
}

// rpcConfig 拦截器配置。
type rpcConfig struct {
	prefix         string
	durationBounds tmetrics.BucketBounds
}

// RPCOption 拦截器可选项。
type RPCOption func(*rpcConfig)

// WithRPCPrefix 覆盖指标名前缀（默认 oj.rpc）。
func WithRPCPrefix(prefix string) RPCOption {
	return func(c *rpcConfig) {
		if p := strings.TrimSuffix(strings.TrimSpace(prefix), "."); p != "" {
			c.prefix = p
		}
	}
}

// WithRPCDurationBounds 覆盖耗时直方图的桶边界（单位必须为秒，见 SecondsBounds）。
func WithRPCDurationBounds(bounds tmetrics.BucketBounds) RPCOption {
	return func(c *rpcConfig) {
		if len(bounds) > 0 {
			c.durationBounds = bounds
		}
	}
}

func newRPCConfig(opts []RPCOption) *rpcConfig {
	cfg := &rpcConfig{
		prefix:         DefaultRPCPrefix,
		durationBounds: DefaultRPCDurationBounds(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// rpcInstrument 持有一次调用要上报的三组指标名。
type rpcInstrument struct {
	requests string
	duration string
	inflight string
}

func newRPCInstrument(cfg *rpcConfig) *rpcInstrument {
	ins := &rpcInstrument{
		requests: cfg.prefix + ".requests",
		duration: cfg.prefix + ".duration.seconds",
		inflight: cfg.prefix + ".inflight.requests",
	}
	// 直方图桶边界只在首次创建该名字时生效，必须在任何上报之前注册，
	// 因此拦截器要在 Sink 初始化（Init / RegisterMetricsSink）之后构建。
	tmetrics.Histogram(ins.duration, cfg.durationBounds)
	return ins
}

// record 上报一次调用的请求数与耗时。
func (ins *rpcInstrument) record(side, service, method, code string, d time.Duration) {
	_ = Report(tmetrics.NewMultiDimensionMetricsX(ins.requests,
		rpcDims("side", side, "service", service, "method", method, "code", code),
		[]*tmetrics.Metrics{tmetrics.NewMetrics(ins.requests, 1, tmetrics.PolicySUM)},
	))
	_ = Report(tmetrics.NewMultiDimensionMetricsX(ins.duration,
		rpcDims("side", side, "service", service, "method", method),
		[]*tmetrics.Metrics{tmetrics.NewMetrics(ins.duration, d.Seconds(), tmetrics.PolicyHistogram)},
	))
}

// setInflight 设置当前并发数。
func (ins *rpcInstrument) setInflight(side, service string, v float64) {
	_ = Report(tmetrics.NewMultiDimensionMetricsX(ins.inflight,
		rpcDims("side", side, "service", service),
		[]*tmetrics.Metrics{tmetrics.NewMetrics(ins.inflight, v, tmetrics.PolicySET)},
	))
}

// rpcDims 按 name/value 交替入参构造维度。
func rpcDims(kv ...string) []*tmetrics.Dimension {
	dims := make([]*tmetrics.Dimension, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		dims = append(dims, &tmetrics.Dimension{Name: kv[i], Value: kv[i+1]})
	}
	return dims
}

// rpcTarget 解析被调服务名与被调方法名。
//
// 优先使用 RPC 名（/trpc.<app>.<server>.<service>/<Method>）：客户端由生成代码写入
// ClientRPCName，服务端在收到请求时写入 ServerRPCName。RPC 名缺失（其他协议或
// 自定义 stub）时回退到消息里的 CalleeServiceName/CalleeMethod。
func rpcTarget(msg codec.Msg) (service, method string) {
	if msg == nil {
		return rpcUnknown, rpcUnknown
	}
	name := msg.ServerRPCName()
	if name == "" {
		name = msg.ClientRPCName()
	}
	service, method = parseRPCName(name)
	if service == rpcUnknown {
		if svc := msg.CalleeServiceName(); svc != "" {
			service = svc
		}
	}
	if method == rpcUnknown {
		if m := msg.CalleeMethod(); m != "" {
			method = m
		}
	}
	return service, method
}

// parseRPCName 解析 RPC 名，形如 /trpc.<app>.<server>.<service>/<Method>。
func parseRPCName(name string) (service, method string) {
	if name == "" {
		return rpcUnknown, rpcUnknown
	}
	parts := strings.SplitN(strings.TrimPrefix(name, "/"), "/", 2)
	service = parts[0]
	if service == "" {
		service = rpcUnknown
	}
	method = rpcUnknown
	if len(parts) == 2 && parts[1] != "" {
		method = parts[1]
	}
	return service, method
}

// rpcCodeLabel 把错误映射为低基数 label。
//
// 成功为 ok；trpc 错误码取去掉 TRPC_ 前缀的小写短名（如 server_system_err）；
// 非 trpc 错误统一为 invoke_unknown_err（errs.Code 对普通 error 返回 RetUnknown）。
func rpcCodeLabel(err error) string {
	if err == nil {
		return rpcCodeOK
	}
	code := errs.Code(err)
	if code == errs.RetOK {
		return rpcCodeOK
	}
	return strings.ToLower(strings.TrimPrefix(code.String(), "TRPC_"))
}

// ServerFilter 返回服务端指标拦截器：上报入向请求数、耗时与并发数。
//
// 拦截器不吞 panic：记为 code=panic 后继续向上抛出，由 recovery filter 处理。
// 在 trpc_go.yaml 的 filter 段中若本拦截器排在 recovery 之前（外层），panic 会先被
// 本拦截器看到；排在 recovery 之后（内层）也能通过自己的 defer 记录到 panic。
func ServerFilter(opts ...RPCOption) filter.ServerFilter {
	return serverFilter(newRPCInstrument(newRPCConfig(opts)))
}

// ClientFilter 返回客户端指标拦截器：上报出向请求数、耗时与并发数。
func ClientFilter(opts ...RPCOption) filter.ClientFilter {
	return clientFilter(newRPCInstrument(newRPCConfig(opts)))
}

func serverFilter(ins *rpcInstrument) filter.ServerFilter {
	return func(ctx context.Context, req interface{},
		next filter.ServerHandleFunc) (rsp interface{}, err error) {
		service, method := rpcTarget(codec.Message(ctx))
		ins.setInflight(rpcSideServer, service, 1)
		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				ins.setInflight(rpcSideServer, service, 0)
				ins.record(rpcSideServer, service, method, rpcCodePanic, time.Since(start))
				panic(p)
			}
			ins.setInflight(rpcSideServer, service, 0)
			ins.record(rpcSideServer, service, method, rpcCodeLabel(err), time.Since(start))
		}()
		return next(ctx, req)
	}
}

func clientFilter(ins *rpcInstrument) filter.ClientFilter {
	return func(ctx context.Context, req, rsp interface{},
		next filter.ClientHandleFunc) (err error) {
		service, method := rpcTarget(codec.Message(ctx))
		ins.setInflight(rpcSideClient, service, 1)
		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				ins.setInflight(rpcSideClient, service, 0)
				ins.record(rpcSideClient, service, method, rpcCodePanic, time.Since(start))
				panic(p)
			}
			ins.setInflight(rpcSideClient, service, 0)
			ins.record(rpcSideClient, service, method, rpcCodeLabel(err), time.Since(start))
		}()
		return next(ctx, req, rsp)
	}
}

// RegisterRPCFilter 以 name 注册 server/client 指标拦截器，供 trpc_go.yaml 的 filter
// 段按名字引用（写作一条 rpcmetrics）。必须在 server/client 构建之前调用。
func RegisterRPCFilter(name string, opts ...RPCOption) {
	ins := newRPCInstrument(newRPCConfig(opts))
	filter.Register(name, serverFilter(ins), clientFilter(ins))
}
