// Package output renders results to the terminal: human-friendly tables
// by default, plus --json / --yaml / --template / --jq for automation.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// Options is wired to root flags: --json / --yaml / --template / --jq.
type Options struct {
	JSON     bool
	YAML     bool
	Template string
	JQ       string
}

// Mode returns a short string used for branching.
func (o Options) Mode() string {
	switch {
	case o.JSON:
		return "json"
	case o.YAML:
		return "yaml"
	case o.Template != "":
		return "template"
	case o.JQ != "":
		return "jq"
	default:
		return "human"
	}
}

// Writer renders a value. Human rendering falls back to table mode for
// slices of structs; single structs fall back to key: value pairs.
type Writer struct {
	W    io.Writer
	Opts Options
}

// New creates a Writer on stdout.
func New(opts Options) *Writer {
	return &Writer{W: os.Stdout, Opts: opts}
}

// Render picks a format based on Options and writes to W.
// `columns` and `rows` are only consulted for human output.
func (w *Writer) Render(value any, columns []string, rows [][]string) error {
	switch w.Opts.Mode() {
	case "json":
		return writeJSON(w.W, value)
	case "yaml":
		return writeYAML(w.W, value)
	case "template":
		return writeTemplate(w.W, value, w.Opts.Template)
	case "jq":
		return writeJQ(w.W, value, w.Opts.JQ)
	default:
		return writeTable(w.W, columns, rows)
	}
}

// RenderValue renders a single value without a table fallback.
func (w *Writer) RenderValue(value any) error {
	switch w.Opts.Mode() {
	case "json":
		return writeJSON(w.W, value)
	case "yaml":
		return writeYAML(w.W, value)
	case "template":
		return writeTemplate(w.W, value, w.Opts.Template)
	case "jq":
		return writeJQ(w.W, value, w.Opts.JQ)
	default:
		return writeHumanStruct(w.W, value)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(v)
}

func writeTemplate(w io.Writer, v any, tmpl string) error {
	t, err := template.New("out").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, v)
}

func writeJQ(w io.Writer, v any, q string) error {
	parsed, err := gojq.Parse(q)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}
	// Roundtrip through JSON so gojq sees plain maps/slices.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	var decoded any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		return err
	}
	iter := parsed.Run(decoded)
	for {
		next, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, ok := next.(error); ok {
			return err
		}
		switch s := next.(type) {
		case string:
			fmt.Fprintln(w, s)
		default:
			data, _ := json.Marshal(next)
			fmt.Fprintln(w, string(data))
		}
	}
}

func writeTable(w io.Writer, cols []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(cols) > 0 {
		fmt.Fprintln(tw, strings.Join(cols, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	return tw.Flush()
}

func writeHumanStruct(w io.Writer, v any) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		_, err := fmt.Fprintln(w, v)
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" {
			name = f.Name
		} else {
			name = strings.SplitN(name, ",", 2)[0]
		}
		val := rv.Field(i).Interface()
		fmt.Fprintf(tw, "%s:\t%v\n", name, val)
	}
	return tw.Flush()
}
