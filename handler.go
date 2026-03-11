// Package sloglambda is a [slog.Handler] for AWS Lambda functions.
//
// It auto-configures log format (JSON or text) and level from Lambda's
// advanced logging environment variables. Options override env values.
//
// See https://docs.aws.amazon.com/lambda/latest/dg/monitoring-logs.html
package sloglambda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
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

	traceLevelDebugOffset = 4
	fatalLevelErrorOffset = 4
)

var (
	kLambdaRecord          = "record"
	kLambdaFunctionName    = "functionName"
	kLambdaFunctionVersion = "version"
	kLambdaRequestId       = "requestId"
	kLambdaLogType         = "type"
)

// Handler implements [slog.Handler] and writes structured log records to an
// [io.Writer]. Use [NewHandler] to create one.
type Handler struct {
	out         io.Writer
	logType     string
	mu          *sync.Mutex
	level       slog.Leveler
	json        bool
	source      bool
	excludeTime bool
	gattr       []groupOrAttrs
}

// Option configures a [Handler]. Pass options to [NewHandler].
type Option func(*Handler)

// WithLevel sets the minimum log level. Messages below this level are dropped.
// Defaults to the value of AWS_LAMBDA_LOG_LEVEL, or INFO if unset.
func WithLevel(level slog.Leveler) Option {
	return func(h *Handler) {
		h.level = level
	}
}

// WithJSON sets the output format to JSON.
func WithJSON() Option {
	return func(h *Handler) {
		h.json = true
	}
}

// WithText sets the output format to key=value text.
func WithText() Option {
	return func(h *Handler) {
		h.json = false
	}
}

// WithSource adds caller information (function, file, line) to each log record.
func WithSource() Option {
	return func(h *Handler) {
		h.source = true
	}
}

// WithType sets the "type" field included in every log record.
// Defaults to "app.log".
func WithType(logType string) Option {
	return func(h *Handler) {
		h.logType = logType
	}
}

// WithoutTime omits the timestamp from log output. Useful for testing or when
// CloudWatch already provides timestamps.
func WithoutTime() Option {
	return func(h *Handler) {
		h.excludeTime = true
	}
}

// NewHandler returns a [Handler] that writes to w.
//
// By default it reads AWS_LAMBDA_LOG_LEVEL (TRACE, DEBUG, INFO, WARN, ERROR,
// FATAL) and AWS_LAMBDA_LOG_FORMAT (json, text) from the environment. Any
// provided options take precedence over environment values.
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
	return loggerLevelFromString(os.Getenv(lambdaEnvLogLevel))
}

func loggerLevelFromString(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return slog.LevelDebug - traceLevelDebugOffset
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "fatal":
		return slog.LevelError + fatalLevelErrorOffset
	default:
		return slog.LevelInfo
	}
}

func lambdaLoggerLevelString(l slog.Level) string {
	str := func(base string, val slog.Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}

	switch {
	case l < slog.LevelDebug:
		return str("TRACE", l-(slog.LevelDebug-traceLevelDebugOffset))
	case l < slog.LevelInfo:
		return str("DEBUG", l-slog.LevelDebug)
	case l < slog.LevelWarn:
		return str("INFO", l-slog.LevelInfo)
	case l < slog.LevelError:
		return str("WARN", l-slog.LevelWarn)
	case l < slog.LevelError+fatalLevelErrorOffset:
		return str("ERROR", l-slog.LevelError)
	default:
		return str("FATAL", l-(slog.LevelError+fatalLevelErrorOffset))
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

	value.append(slog.String(slog.LevelKey, lambdaLoggerLevelString(record.Level)))
	value.append(slog.String(slog.MessageKey, record.Message))

	if !record.Time.IsZero() && !h.excludeTime {
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

		value.append(slog.Group(slog.SourceKey,
			slog.String("function", frame.Function),
			slog.String("file", frame.File),
			slog.Int("line", frame.Line),
		))
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

	buf := getBuffer()
	defer putBuffer(buf)

	if h.json {
		if err := json.NewEncoder(buf).Encode(topLevel); err != nil {
			h.mu.Lock()
			defer h.mu.Unlock()

			fmt.Fprintf(h.out, `{"level":"ERROR","msg":"failed to encode log record: %v"}`, err)
			fmt.Fprintln(h.out)
			return err
		}
	} else {
		if err := writeTextRecord(buf, topLevel, ""); err != nil {
			h.mu.Lock()
			defer h.mu.Unlock()

			fmt.Fprintf(h.out, `level=ERROR msg="failed to encode log record: %v"`, err)
			fmt.Fprintln(h.out)
			return err
		}
		// Remove the last trailing space
		buf.Truncate(buf.Len() - 1)
		buf.Write([]byte("\n"))
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := io.Copy(h.out, buf)
	return err
}

var _ slog.Handler = (*Handler)(nil)

type logRecord map[string]any

func (r logRecord) append(attr slog.Attr) {
	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
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
	} else {
		r[attr.Key] = normalizeValue(attr.Value)
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

func (r logRecord) keys() []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	return keys
}

var bufferPool = sync.Pool{
	New: func() any {
		b := bytes.NewBuffer(nil)
		b.Grow(1024)
		return b
	},
}

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(b *bytes.Buffer) {
	const maxBufferSize = 16 << 10

	if b.Cap() <= maxBufferSize {
		b.Reset()
		bufferPool.Put(b)
	}
}

type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func writeTextRecord(w io.Writer, record logRecord, path string) error {
	if record == nil {
		return nil
	}

	keys := record.keys()
	slices.Sort(keys)

	for _, key := range keys {
		value := record[key]
		if path != "" {
			key = path + "." + key
		}

		if _, ok := value.(logRecord); !ok {
			w.Write([]byte(key))
			w.Write([]byte("="))
		}

		switch v := value.(type) {
		case logRecord:
			if err := writeTextRecord(w, v, key); err != nil {
				return err
			}
		case string:
			w.Write([]byte(strconv.Quote(v)))
		case fmt.Stringer:
			// This is here because nilaway can't figure out that v is not nil
			if v != nil {
				w.Write([]byte(v.String()))
			}
		default:
			fmt.Fprintf(w, "%v", v)
		}

		if _, ok := value.(logRecord); !ok {
			w.Write([]byte(" "))
		}
	}

	return nil
}

func normalizeValue(v slog.Value) any {
	switch v.Kind() {
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindString:
		return v.String()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindLogValuer:
		return normalizeValue(v.Resolve())
	case slog.KindAny:
		return normalizeAnyValue(v.Any())
	case slog.KindGroup:
		attrs := v.Group()
		if len(attrs) == 0 {
			return nil
		}
		rec := make(logRecord, len(attrs))
		for _, a := range attrs {
			rec.append(a)
		}
		return rec
	default:
		return v.String()
	}
}

func normalizeAnyValue(val any) any {
	switch v := val.(type) {
	case error:
		return v.Error()
	case json.Marshaler:
		b, err := v.MarshalJSON()
		if err != nil {
			return err.Error()
		}
		return string(b)
	default:
		return val
	}
}
