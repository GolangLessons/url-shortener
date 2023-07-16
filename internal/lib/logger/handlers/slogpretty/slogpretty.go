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
		fieldsFormat  FieldsFormat // Json, JsonIndent, Yaml
		useLevelEmoji bool         // the flag instructs to use emoji in the logging level text
	}
	FieldsFormat string
	Record       = slog.Record
	Attr         = slog.Attr
	SlogOpts     = slog.HandlerOptions
	SlogHandler  = slog.Handler
	marshalFunc  func(any) ([]byte, error)
	Level        = slog.Level
	levelInfo    struct {
		text      string
		emoji     string
		colorFunc func(format string, a ...interface{}) string
	}
)

const (
	ffJson       FieldsFormat = "json"
	ffJsonIndent FieldsFormat = "json-indent"
	ffYaml       FieldsFormat = "yaml"
)

var (
	fieldFormats = map[FieldsFormat]marshalFunc{
		ffJson: json.Marshal,
		ffJsonIndent: func(v any) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		},
		ffYaml: yaml.Marshal,
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
		fieldsFormat: ffJson,
		SlogOpts:     SlogOpts{Level: slog.LevelDebug},
	}
}

func (f FieldsFormat) Validate() error {
	if _, ok := fieldFormats[f]; !ok {
		xs := make([]string, 0, len(fieldFormats))
		for ff := range fieldFormats {
			xs = append(xs, string(ff))
		}
		return fmt.Errorf("invalid fields format %q, must be one of %s", f, strings.Join(xs, ","))
	}
	return nil
}

func (f FieldsFormat) Marshal(v any) ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return fieldFormats[f](v)
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

func (h Handler) WithAddSource(v bool) Handler {
	h.SlogOpts.AddSource = v
	return h
}

func (h Handler) WithReplaceAttr(replaceAttr func(groups []string, a Attr) Attr) Handler {
	h.SlogOpts.ReplaceAttr = replaceAttr
	return h
}

func (h Handler) WithFieldsFormat(f FieldsFormat) Handler {
	h.fieldsFormat = f
	return h
}

func (h Handler) WithLevelEmoji(v bool) Handler {
	h.useLevelEmoji = v
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
	s, err := h.fieldsFormat.Marshal(xs)
	if err != nil {
		return "", err
	}
	return color.WhiteString(string(s)), nil
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
