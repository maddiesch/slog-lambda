package sloglambda

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

const (
	lambdaEnvLogLevel        = "AWS_LAMBDA_LOG_LEVEL"
	lambdaEnvLogFormat       = "AWS_LAMBDA_LOG_FORMAT"
	lambdaEnvFunctionName    = "AWS_LAMBDA_FUNCTION_NAME"
	lambdaEnvFunctionVersion = "AWS_LAMBDA_FUNCTION_VERSION"
)

var (
	kLambdaRecord          = "record"
	kLambdaFunctionName    = "functionName"
	kLambdaFunctionVersion = "functionVersion"
	kLambdaRequestId       = "requestId"
	kLambdaLogType         = "type"
)

type Handler struct {
	out     io.Writer
	logType string
	mu      *sync.Mutex
	level   slog.Level
	json    bool
	source  bool
	gattr   []groupOrAttrs
}

type Option func(*Handler)

func WithLevel(level slog.Level) Option {
	return func(h *Handler) {
		h.level = level
	}
}

func WithJSON() Option {
	return func(h *Handler) {
		h.json = true
	}
}

func WithText() Option {
	return func(h *Handler) {
		h.json = false
	}
}

func WithSource() Option {
	return func(h *Handler) {
		h.source = true
	}
}

func WithType(logType string) Option {
	return func(h *Handler) {
		h.logType = logType
	}
}

func NewHandler(w io.Writer, options ...Option) *Handler {
	h := &Handler{
		out:     w,
		mu:      new(sync.Mutex),
		level:   loggerLevelFromLambdaEnv(),
		json:    loggerIsJSON(),
		source:  false,
		logType: "app.log",
	}

	for _, opt := range options {
		opt(h)
	}

	return h
}

func loggerLevelFromLambdaEnv() slog.Level {
	env := os.Getenv(lambdaEnvLogLevel)
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "trace":
		return slog.LevelDebug - 4
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "fatal":
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
}

func loggerIsJSON() bool {
	env := os.Getenv(lambdaEnvLogFormat)
	return strings.ToLower(strings.TrimSpace(env)) == "json"
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) WithAttrs(attr []slog.Attr) slog.Handler {
	return h.copy(groupOrAttrs{attrs: attr})
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h.copy(groupOrAttrs{group: name})
}

func (h *Handler) copy(g groupOrAttrs) *Handler {
	c := *h
	c.gattr = make([]groupOrAttrs, len(h.gattr)+1)
	copy(c.gattr, h.gattr)
	c.gattr[len(c.gattr)-1] = g
	return &c
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	value := make(logRecord, 10)
	topLevel := value

	value.append(slog.Any(slog.LevelKey, record.Level))
	value.append(slog.String(slog.MessageKey, record.Message))

	if !record.Time.IsZero() {
		value.append(slog.Time(slog.TimeKey, record.Time))
	}

	lambdaGroup := make(logRecord, 3)
	if value, ok := os.LookupEnv(lambdaEnvFunctionName); ok {
		lambdaGroup.append(slog.String(kLambdaFunctionName, value))
	}
	if value, ok := os.LookupEnv(lambdaEnvFunctionVersion); ok {
		lambdaGroup.append(slog.String(kLambdaFunctionVersion, value))
	}

	if lc, _ := lambdacontext.FromContext(ctx); lc != nil {
		lambdaGroup.append(slog.String(kLambdaRequestId, lc.AwsRequestID))
	}

	if len(lambdaGroup) > 0 {
		value[kLambdaRecord] = lambdaGroup
	}

	if h.logType != "" {
		value[kLambdaLogType] = h.logType
	}

	if record.PC != 0 && h.source {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		frame, _ := frames.Next()
		value[slog.SourceKey] = &slog.Source{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		}
	}

	gattr := h.gattr
	if record.NumAttrs() == 0 {
		for len(gattr) > 0 && gattr[len(gattr)-1].group != "" {
			gattr = gattr[:len(gattr)-1]
		}
	}

	for _, ga := range gattr {
		if ga.group == "" {
			for _, a := range ga.attrs {
				value.append(a)
			}
		} else {
			group := make(logRecord, 10)
			value[ga.group] = group
			value = group
		}
	}

	record.Attrs(func(a slog.Attr) bool {
		value.append(a)
		return true
	})

	topLevel.clean()

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.json {
		return json.NewEncoder(h.out).Encode(topLevel)
	}

	panic("text handler support is not currently implemented")
}

var _ slog.Handler = (*Handler)(nil)

type logRecord map[string]any

func (r logRecord) append(attr slog.Attr) {
	attr.Value = attr.Value.Resolve()

	if attr.Equal(slog.Attr{}) {
		return
	}

	switch attr.Value.Kind() {
	case slog.KindTime:
		r[attr.Key] = attr.Value.Time().Format(time.RFC3339)
	case slog.KindGroup:
		group := attr.Value.Group()
		if len(group) == 0 {
			return
		}

		if attr.Key == "" {
			for _, a := range group {
				r.append(a)
			}
		} else {
			r[attr.Key] = make(logRecord, len(group))
			for _, a := range group {
				r[attr.Key].(logRecord).append(a)
			}
		}
	default:
		r[attr.Key] = attr.Value.Any()
	}
}

func (r logRecord) clean() {
	for k, v := range r {
		if lr, ok := v.(logRecord); ok {
			if len(lr) == 0 {
				delete(r, k)
			} else {
				lr.clean()
			}
		}
	}
}

type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}
