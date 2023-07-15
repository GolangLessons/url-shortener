package slogpretty

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
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
		timeLayout    string // by default, do not display the time locally
		attrs         []Attr
		groups        []string
		fieldsFormat  string // fieldsFormatJson, fieldsFormatJsonIndent, fieldsFormatYaml
		useLevelEmoji bool   // the flag instructs to use emoji in the logging level text
	}
	Record      = slog.Record
	Attr        = slog.Attr
	SlogOpts    = slog.HandlerOptions
	SlogHandler = slog.Handler
	marshalFunc func(any) ([]byte, error)
	Level       = slog.Level
	levelInfo   struct {
		text      string
		emoji     string
		colorFunc func(format string, a ...interface{}) string
	}
)

const (
	fieldsFormatJson       = "json"
	fieldsFormatJsonIndent = "json-indent"
	fieldsFormatYaml       = "yaml"
)

var (
	marshalers = map[string]marshalFunc{
		fieldsFormatJson: json.Marshal,
		fieldsFormatJsonIndent: func(v any) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		},
		fieldsFormatYaml: yaml.Marshal,
	}
	levelsInfo = map[Level]levelInfo{
		slog.LevelDebug: {
			"DEBUG", "👀", color.MagentaString},
		slog.LevelInfo: {
			"INFO ", "✅", color.BlueString},
		slog.LevelWarn: {
			"WARN ", "🔥", color.YellowString},
		slog.LevelError: {
			"ERROR", "❌", color.RedString},
	}
)

func NewHandler() Handler {
	return Handler{
		logger:       log.New(os.Stderr, "", 0),
		fieldsFormat: fieldsFormatJson,
		SlogOpts:     SlogOpts{Level: slog.LevelDebug},
	}
}

func (h Handler) WithOutput(output io.Writer) Handler {
	h.logger = log.New(output, "", 0)
	return h
}

func (h Handler) WithTimeLayout(layout string) Handler {
	h.timeLayout = layout
	return h
}

func (h Handler) WithLevel(l Level) Handler {
	h.SlogOpts.Level = l
	return h
}

func (h Handler) WithSlogOpts(o SlogOpts) Handler {
	h.SlogOpts = o
	return h
}

func (h Handler) WithAddSource() Handler {
	h.SlogOpts.AddSource = true
	return h
}

func (h Handler) WithReplaceAttr(replaceAttr func(groups []string, a Attr) Attr) Handler {
	h.SlogOpts.ReplaceAttr = replaceAttr
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

func (h Handler) WithLevelEmoji() Handler {
	h.useLevelEmoji = true
	return h
}

func (h Handler) Handle(_ context.Context, r Record) error {
	var outputParts []interface{}
	if h.timeLayout != "" {
		outputParts = append(outputParts, color.WhiteString(r.Time.Format(h.timeLayout)))
	}

	outputParts = append(outputParts, h.recordLevel(r), color.CyanString(r.Message))

	strAttrs, err := h.recordAttrs(r)
	if err != nil {
		return err
	}
	if strAttrs != "" {
		outputParts = append(outputParts, strAttrs)
	}

	if h.SlogOpts.AddSource {
		outputParts = append(outputParts, color.GreenString(recordFormatSource(r)))
	}

	h.logger.Println(outputParts...)

	return nil
}

func (h Handler) Enabled(_ context.Context, l Level) bool {
	return l.Level() >= h.SlogOpts.Level.Level()
}

func (h Handler) WithAttrs(attrs []Attr) SlogHandler {
	h.attrs = append(h.attrs, attrs...)
	return h
}

func (h Handler) WithGroup(name string) SlogHandler {
	h.groups = append(h.groups, name)
	return h
}

func (h Handler) recordAttrs(r Record) (string, error) {
	xs := attrsValues(append(recordAttrs(r), h.attrs...)...)
	if len(xs) == 0 {
		return "", nil
	}
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
		return "", err
	}
	return s, nil
}

func (h Handler) recordLevel(r Record) string {
	l := levelsInfo[r.Level.Level()]
	level := l.text
	if level == "" {
		level = r.Level.String()
	}
	if l.colorFunc != nil {
		level = l.colorFunc(level)
	}
	if h.useLevelEmoji && l.emoji != "" {
		level = l.emoji + " " + level
	}
	return level
}

// formats a Source for the log event.
func recordFormatSource(r Record) string {
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

func (f marshalFunc) formatFields(fields map[string]interface{}) (string, error) {
	b, err := f(fields)
	if err != nil {
		return "", err
	}
	return color.WhiteString(strings.TrimSpace(string(b))), nil
}

func recordAttrs(r Record) []Attr {
	xs := make([]Attr, 0, r.NumAttrs())
	r.Attrs(func(a Attr) bool {
		xs = append(xs, a)
		return true
	})
	return xs
}

func attrsValues(attrs ...Attr) map[string]interface{} {
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
