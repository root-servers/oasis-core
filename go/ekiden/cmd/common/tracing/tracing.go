// Package tracing implements a tracing service.
package tracing

import (
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	"github.com/oasislabs/ekiden/go/common/service"
)

const (
	cfgTracingEnabled                    = "tracing.enabled"
	cfgTracingReporterFlushInterval      = "tracing.reporter.flush_interval"
	cfgTracingReporterLocalAgentHostPort = "tracing.reporter.agent_addr"
	cfgTracingSamplerParam               = "tracing.sampler.param"
)

var (
	tracingEnabled                    bool
	tracingReporterFlushInterval      time.Duration
	tracingReporterLocalAgentHostPort string
	tracingSamplerParam               float64
)

type tracingService struct {
	closer io.Closer
}

func (svc *tracingService) Cleanup() {
	if svc.closer != nil {
		svc.closer.Close()
		svc.closer = nil
	}
}

// New constructs a new tracing service.
func New(cmd *cobra.Command, serviceName string) (service.CleanupAble, error) {
	enabled, _ := cmd.Flags().GetBool(cfgTracingEnabled)
	reporterFlushInterval, _ := cmd.Flags().GetDuration(cfgTracingReporterFlushInterval)
	reporterLocalAgentHostPort, _ := cmd.Flags().GetString(cfgTracingReporterLocalAgentHostPort)
	samplerParam, _ := cmd.Flags().GetFloat64(cfgTracingSamplerParam)

	cfg := config.Configuration{
		Disabled: !enabled,
		Reporter: &config.ReporterConfig{
			BufferFlushInterval: reporterFlushInterval,
			LocalAgentHostPort:  reporterLocalAgentHostPort,
		},
		Sampler: &config.SamplerConfig{
			Param: samplerParam,
		},
	}

	closer, err := cfg.InitGlobalTracer(serviceName, config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}

	return &tracingService{closer: closer}, nil
}

// RegisterFlags registers the flags used by the tracing service.
func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&tracingEnabled, cfgTracingEnabled, true, "Enable tracing")
	cmd.Flags().DurationVar(&tracingReporterFlushInterval, cfgTracingReporterFlushInterval, 1*time.Second, "How often the buffer is force-flushed, even if it's not full")
	cmd.Flags().StringVar(&tracingReporterLocalAgentHostPort, cfgTracingReporterLocalAgentHostPort, "localhost:6831", "Send spans to jaeger-agent at this address")
	cmd.Flags().Float64Var(&tracingSamplerParam, cfgTracingSamplerParam, 0.001, "Probability for probabilistic sampler")

	for _, v := range []string{
		cfgTracingEnabled,
		cfgTracingReporterFlushInterval,
		cfgTracingReporterLocalAgentHostPort,
		cfgTracingSamplerParam,
	} {
		_ = viper.BindPFlag(v, cmd.Flags().Lookup(v))
	}
}