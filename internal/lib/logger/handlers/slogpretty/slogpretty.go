package slogpretty

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/exp/slog"
)

type (
	Handler struct {
		SlogOpts
		logger        *log.Logger
		attrs         []slog.Attr
		groups        []string
		fieldsFormat  string // fieldsFormatJson, fieldsFormatJsonIndent, fieldsFormatYaml
		useLevelEmoji bool
	}
	SlogOpts    = slog.HandlerOptions
	marshalFunc func(any) ([]byte, error)
)

const (
	fieldsFormatJson       = "json"
	fieldsFormatJsonIndent = "json-indent"
	fieldsFormatYaml       = "yaml"
)

var marshalers = map[string]marshalFunc{
	fieldsFormatJson: json.Marshal,
	fieldsFormatJsonIndent: func(v any) ([]byte, error) {
		return json.MarshalIndent(v, "", "  ")
	},
	fieldsFormatYaml: yaml.Marshal,
}

func NewHandler(out io.Writer) Handler {
	return Handler{
		logger:       log.New(out, "", 0),
		fieldsFormat: fieldsFormatJson,
		SlogOpts:     SlogOpts{Level: slog.LevelDebug},
	}
}

func (h Handler) WithLevel(l slog.Level) Handler {
	h.SlogOpts.Level = l
	return h
}

func (h Handler) WithSlogOpts(o SlogOpts) Handler {
	h.SlogOpts = o
	return h
}

func (h Handler) WithFieldsFormatYaml() Handler {
	h.fieldsFormat = fieldsFormatYaml
	return h
}

func (h Handler) WithFieldsFormatJsonIndent() Handler {
	h.fieldsFormat = fieldsFormatJsonIndent
	return h
}

func (h Handler) WithUseLevelEmoji() Handler {
	h.useLevelEmoji = true
	return h
}

func (h Handler) Handle(_ context.Context, r slog.Record) error {
	_println := []interface{}{h.lev(r), color.CyanString(r.Message)}

	if xs := attrsValues(append(recordAttrs(r), h.attrs...)...); len(xs) != 0 {
		for i := len(h.groups) - 1; i >= 0; i-- {
			xs = map[string]interface{}{
				h.groups[i]: xs,
			}
		}
		marshaler, ok := marshalers[h.fieldsFormat]
		if !ok {
			marshaler = json.Marshal
		}
		s, err := marshaler.formatFields(xs)
		if err != nil {
			return err
		}
		_println = append(_println, s)
	}

	if h.SlogOpts.AddSource {
		_println = append(_println, color.GreenString(recordFormatSource(r)))
	}

	h.logger.Println(_println...)

	return nil
}

func (h Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l.Level() >= h.SlogOpts.Level.Level()
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return h
}

func (h Handler) WithGroup(name string) slog.Handler {
	h.groups = append(h.groups, name)
	return h
}

// formats a Source for the log event.
func recordFormatSource(r slog.Record) string {
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()

	function := filepath.Base(f.Function)
	for i, ch := range function {
		if string(ch) == "." {
			function = function[i:]
			break
		}
	}
	return fmt.Sprintf("%s:%d%s", filepath.Base(f.File), f.Line, function)
}

func (h Handler) lev(r slog.Record) string {
	level, emoji := r.Level.String(), ""
	switch r.Level {
	case slog.LevelDebug:
		emoji = "👀"
		level = color.MagentaString(
			"DEBUG")
	case slog.LevelInfo:
		emoji = "✅"
		level = color.BlueString(
			"INFO ")
	case slog.LevelWarn:
		emoji = "🔥"
		level = color.YellowString(
			"WARN ")
	case slog.LevelError:
		emoji = "❌"
		level = color.RedString(
			"ERROR")
	}
	if h.useLevelEmoji && emoji != "" {
		return emoji + " " + level
	}
	return level
}

func (f marshalFunc) formatFields(fields map[string]interface{}) (string, error) {
	var (
		b   []byte
		err error
	)
	b, err = f(fields)
	if err != nil {
		return "", err
	}
	return color.WhiteString(strings.TrimSpace(string(b))), nil
}

func recordAttrs(r slog.Record) []slog.Attr {
	xs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		xs = append(xs, a)
		return true
	})
	return xs
}

func attrsValues(attrs ...slog.Attr) map[string]interface{} {
	fields := make(map[string]interface{}, len(attrs))
	for _, a := range attrs {
		if a.Value.Kind() == slog.KindGroup {
			fields[a.Key] = attrsValues(a.Value.Group()...)
		} else {
			fields[a.Key] = a.Value.Any()
		}
	}
	return fields
}
