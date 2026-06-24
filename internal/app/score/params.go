package score

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// listStyle selects the pagination flag/param naming convention. The two
// backends differ: Sanr uses take/skip(/attributes); Arena uses limit/offset.
type listStyle int

const (
	styleSanr listStyle = iota
	styleArena
)

// queryFlags holds the standard list-query flags plus an escape hatch for
// endpoint-specific query parameters. It is populated into any generated
// *Params struct via a JSON round-trip (the structs carry matching json tags).
type queryFlags struct {
	style  listStyle
	page   int // take (Sanr) or limit (Arena)
	off    int // skip (Sanr) or offset (Arena)
	sort   string
	filter string
	attrs  string
	params []string // repeatable key=value for endpoint-specific params
}

// bindList registers the list-query flag set for the given style and page size.
func (q *queryFlags) bindList(fs *pflag.FlagSet, style listStyle, defaultPage int) {
	q.style = style
	if style == styleArena {
		fs.IntVar(&q.page, "limit", defaultPage, "page size")
		fs.IntVar(&q.off, "offset", 0, "records to skip")
	} else {
		fs.IntVar(&q.page, "take", defaultPage, "page size (1..500)")
		fs.IntVar(&q.off, "skip", 0, "offset for pagination")
		fs.StringVar(&q.attrs, "attributes", "", `quoted attribute list, e.g. "id","asset"`)
	}
	fs.StringVar(&q.sort, "sort", "", "sort spec, e.g. createdAt:desc,verifiedAt:asc")
	fs.StringVar(&q.filter, "filter", "", `filter JSON without outer braces, e.g. "asset":"BTC"`)
	q.bindParams(fs)
}

// bindParams registers only the generic --param escape hatch (for endpoints
// that are not list-style but still accept query parameters).
func (q *queryFlags) bindParams(fs *pflag.FlagSet) {
	fs.StringArrayVar(&q.params, "param", nil, "extra query param key=value (repeatable; value parsed as JSON when possible)")
}

// into populates target (a pointer to a generated *Params struct) from the flags.
func (q *queryFlags) into(target any) error {
	m := map[string]any{}
	if q.style == styleArena {
		if q.page > 0 {
			m["limit"] = q.page
		}
		if q.off > 0 {
			m["offset"] = q.off
		}
	} else {
		if q.page > 0 {
			m["take"] = q.page
		}
		if q.off > 0 {
			m["skip"] = q.off
		}
		if q.attrs != "" {
			m["attributes"] = q.attrs
		}
	}
	if q.sort != "" {
		m["sort"] = q.sort
	}
	if q.filter != "" {
		m["filter"] = q.filter
	}
	for _, kv := range q.params {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return usageError(fmt.Sprintf("invalid --param %q, expected key=value", kv))
		}
		m[k] = parseScalar(v)
	}
	return jsonRoundTrip(m, target)
}

// parseScalar interprets a flag value as JSON when it parses (so true/123/[...]
// keep their type), otherwise as a plain string.
func parseScalar(v string) any {
	var parsed any
	if json.Unmarshal([]byte(v), &parsed) == nil {
		return parsed
	}
	return v
}

func jsonRoundTrip(src, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// bodyFlags reads a JSON request body from --data, --data-file, or stdin.
type bodyFlags struct {
	data     string
	dataFile string
}

func (b *bodyFlags) bind(fs *pflag.FlagSet) {
	fs.StringVar(&b.data, "data", "", "request body as a JSON string")
	fs.StringVar(&b.dataFile, "data-file", "", `path to a JSON body file, or "-" for stdin`)
}

// into decodes the request body into target (a generated *JSONRequestBody type).
func (b *bodyFlags) into(cmd *cobra.Command, target any) error {
	raw, err := b.read(cmd)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return usageError("a request body is required; pass --data or --data-file")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return usageError("invalid JSON body: " + err.Error())
	}
	return nil
}

func (b *bodyFlags) read(cmd *cobra.Command) ([]byte, error) {
	switch {
	case b.data != "":
		return []byte(b.data), nil
	case b.dataFile == "-":
		return io.ReadAll(cmd.InOrStdin())
	case b.dataFile != "":
		return os.ReadFile(b.dataFile)
	default:
		return nil, nil
	}
}

// pathInt parses a positional path argument that must be an integer.
func pathInt(arg, name string) (int, error) {
	n, err := strconv.Atoi(arg)
	if err != nil {
		return 0, usageError(fmt.Sprintf("%s must be an integer, got %q", name, arg))
	}
	return n, nil
}
