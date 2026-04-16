// Cross-parser ISO test. Every supported format ships a matching
// fixture under testdata/real_world.* that decodes to the SAME
// normalized map. This test is the regression guard that proves the
// parsers agree on the canonical key/value set regardless of source
// format.
package parser_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kitsunium/sdk/v1/components/config/adapters/args"
	"github.com/kitsunium/sdk/v1/components/config/adapters/env"
	"github.com/kitsunium/sdk/v1/components/config/adapters/ini"
	"github.com/kitsunium/sdk/v1/components/config/adapters/json"
	"github.com/kitsunium/sdk/v1/components/config/adapters/toml"
	"github.com/kitsunium/sdk/v1/components/config/adapters/xml"
	"github.com/kitsunium/sdk/v1/components/config/adapters/yaml"
)

// canonicalRealWorld is the expected normalized map that every
// real_world.<format> fixture must decode to (after stripping any
// format-specific root prefix such as XML's "config.").
//
// Maintain this table in lockstep with every testdata/real_world.*
// file: adding a key here MUST come with an equivalent update to each
// fixture in testdata/.
var canonicalRealWorld = map[string]string{
	"database.host":      "localhost",
	"database.port":      "5432",
	"database.name":      "myapp_production",
	"database.user":      "dbuser",
	"database.pass":      "secret123",
	"database.pool.size": "10",

	"redis.host":            "redis.local",
	"redis.port":            "6379",
	"redis.db":              "0",
	"redis.pass":            "",
	"redis.cluster.enabled": "false",

	"logging.level":       "info",
	"logging.format":      "json",
	"logging.output":      "stdout",
	"logging.file":        "/var/log/app.log",
	"logging.max.size":    "100",
	"logging.max.backups": "3",
	"logging.max.age":     "30",

	"features.new.ui":        "true",
	"features.beta.features": "false",
	"features.experimental":  "false",
	"features.feature.flags": "flag1,flag2,flag3",

	"server.host":             "0.0.0.0",
	"server.port":             "8080",
	"server.read.timeout":     "30",
	"server.write.timeout":    "30",
	"server.idle.timeout":     "120",
	"server.max.header.bytes": "1048576",

	"security.jwt.secret":      "super-secret-key-do-not-share",
	"security.bcrypt.cost":     "10",
	"security.session.timeout": "3600",
	"security.csrf.enabled":    "true",
	"security.cors.origins":    "https://example.com,https://app.example.com",
}

// TestRealWorld_CrossParserISO loads every testdata/real_world.<fmt>
// fixture through its matching parser and asserts each decodes to the
// canonical normalized map. XML's implicit "config." root prefix is
// stripped before comparison; all other formats expose the flat
// namespace directly.
func TestRealWorld_CrossParserISO(t *testing.T) {
	cases := []struct {
		name     string
		decode   func(t *testing.T) map[string]string
		stripPfx string
	}{
		{name: "json", decode: decodeJSONFixture},
		{name: "yaml", decode: decodeYAMLFixture},
		{name: "toml", decode: decodeTOMLFixture},
		{name: "ini", decode: decodeINIFixture},
		{name: "xml", decode: decodeXMLFixture, stripPfx: "config."},
		{name: "env", decode: decodeENVFixture},
		{name: "args", decode: decodeARGSFixture},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripPrefix(tc.decode(t), tc.stripPfx)
			assertSuperset(t, canonicalRealWorld, got)
		})
	}
}

// stripPrefix returns a copy of in with every key's pfx prefix
// removed. Empty pfx returns in unchanged. Keys that do not start
// with pfx are dropped — this is the behaviour we want for the XML
// root element, since anything outside <config> is noise.
func stripPrefix(in map[string]string, pfx string) map[string]string {
	if pfx == "" {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		after, ok := strings.CutPrefix(k, pfx)
		if !ok {
			continue
		}
		out[after] = v
	}
	return out
}

// assertSuperset fails the test when want is NOT a subset of got —
// i.e. every canonical key/value must be present in the parser's
// output. Extra keys in got are tolerated (parsers may surface
// metadata the canonical table does not care about).
func assertSuperset(t *testing.T, want, got map[string]string) {
	t.Helper()
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Errorf("missing key %q (expected %q)", k, wv)
			continue
		}
		if gv != wv {
			t.Errorf("key %q = %q, want %q", k, gv, wv)
		}
	}
}

func decodeJSONFixture(t *testing.T) map[string]string {
	t.Helper()
	m, err := json.NewJSON(filepath.Join("..", "testdata", "real_world.json")).Load()
	if err != nil {
		t.Fatalf("json load: %v", err)
	}
	return m
}

func decodeYAMLFixture(t *testing.T) map[string]string {
	t.Helper()
	m, err := yaml.NewYAML(filepath.Join("..", "testdata", "real_world.yaml")).Load()
	if err != nil {
		t.Fatalf("yaml load: %v", err)
	}
	return m
}

func decodeTOMLFixture(t *testing.T) map[string]string {
	t.Helper()
	m, err := toml.NewTOML(filepath.Join("..", "testdata", "real_world.toml")).Load()
	if err != nil {
		t.Fatalf("toml load: %v", err)
	}
	return m
}

func decodeINIFixture(t *testing.T) map[string]string {
	t.Helper()
	m, err := ini.NewINI(filepath.Join("..", "testdata", "real_world.ini")).Load()
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	return m
}

func decodeXMLFixture(t *testing.T) map[string]string {
	t.Helper()
	m, err := xml.NewXML(filepath.Join("..", "testdata", "real_world.xml")).Load()
	if err != nil {
		t.Fatalf("xml load: %v", err)
	}
	return m
}

// decodeENVFixture reads testdata/real_world_environ.txt, injects
// every KEY=VALUE line into the process environment via t.Setenv
// (auto-cleaned at test end), then invokes the ENV parser with the
// APP_ prefix — matching the fixture's naming.
func decodeENVFixture(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "testdata", "real_world_environ.txt")
	for _, kv := range readFixtureLines(t, path) {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		t.Setenv(kv[:eq], kv[eq+1:])
	}
	m, err := env.NewENV("APP_").Load()
	if err != nil {
		t.Fatalf("env load: %v", err)
	}
	return m
}

// decodeARGSFixture reads testdata/real_world.args line-by-line and
// feeds the collected flags to ARGS.ParseArgs — matches the
// constructor's expected []string shape without touching os.Args.
func decodeARGSFixture(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "testdata", "real_world.args")
	lines := readFixtureLines(t, path)
	m, err := args.NewARGS(false).ParseArgs(lines)
	if err != nil {
		t.Fatalf("args parse: %v", err)
	}
	return m
}

// readFixtureLines returns the non-empty, non-comment lines of path.
// Comments start with '#' or ';' and are stripped; whitespace-only
// lines are ignored.
func readFixtureLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line[0] == '#' || line[0] == ';' {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan fixture %s: %v", path, err)
	}
	return lines
}
