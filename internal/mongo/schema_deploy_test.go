package mongo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type schemaCommandCall struct {
	name  string
	args  []string
	input []byte
}

func schemaTestWrite(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func schemaTestFixture(t *testing.T) (string, string, DeployOptions) {
	t.Helper()
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	schemaTestWrite(t, root, "data/blog-posts.json", `[{"slug":"first"}]`)
	schemaTestWrite(t, root, "schemas/blog-posts.json", `{"bsonType":"object","required":["slug"]}`)
	return root, dataDir, DeployOptions{
		ProjectRoot: root,
		Schemas:     map[string]string{"blog-posts": "schemas/blog-posts.json"},
	}
}

func schemaTestCommands(t *testing.T) (*[]string, *[]schemaCommandCall) {
	t.Helper()
	oldLookPath, oldRunCommand := lookPath, runCommand
	t.Cleanup(func() {
		lookPath, runCommand = oldLookPath, oldRunCommand
	})
	var lookups []string
	var calls []schemaCommandCall
	lookPath = func(name string) (string, error) {
		lookups = append(lookups, name)
		return name, nil
	}
	runCommand = func(name string, args []string, input []byte) error {
		var copiedInput []byte
		if input != nil {
			copiedInput = append([]byte{}, input...)
		}
		calls = append(calls, schemaCommandCall{name, append([]string{}, args...), copiedInput})
		return nil
	}
	return &lookups, &calls
}

func schemaTestArgument(t *testing.T, args []string, key string) string {
	t.Helper()
	for i, arg := range args {
		if arg == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("missing argument %s in %v", key, args)
	return ""
}

func schemaTestHasArgument(args []string, key string) bool {
	for _, arg := range args {
		if arg == key {
			return true
		}
	}
	return false
}

func TestSchemaDeploymentLifecycle(t *testing.T) {
	for _, source := range []struct {
		name      string
		files     map[string]string
		wantInput string
		jsonArray bool
	}{
		{name: "object", files: map[string]string{"blog-posts.json": `{"slug":"first"}`}, wantInput: `{"slug":"first"}`},
		{name: "array", files: map[string]string{"blog-posts.json": `[{"slug":"first"}]`}, wantInput: `[{"slug":"first"}]`, jsonArray: true},
		{name: "directory", files: map[string]string{"blog-posts/02.json": "{\n\"slug\": \"second\"\n}", "blog-posts/01.json": `{"slug":"first"}`}, wantInput: "{\"slug\":\"first\"}\n{\"slug\":\"second\"}\n"},
	} {
		t.Run(source.name, func(t *testing.T) {
			root := t.TempDir()
			dataDir := filepath.Join(root, "data")
			for name, content := range source.files {
				schemaTestWrite(t, dataDir, name, content)
			}
			const schema = `{"bsonType":"object","description":"literal quote: \"; throw Error('injected');//","required":["slug"]}`
			schemaTestWrite(t, root, "schemas/blog.json", schema)
			lookups, calls := schemaTestCommands(t)
			plan, err := PrepareCollection("blog-posts", dataDir, DeployOptions{ProjectRoot: root, Schemas: map[string]string{"blog-posts": "schemas/blog.json"}})
			if err != nil {
				t.Fatalf("PrepareCollection: %v", err)
			}
			if len(*calls) != 0 {
				t.Fatalf("preparation ran a process: %+v", *calls)
			}
			for _, name := range []string{"mongosh", "mongoimport"} {
				if !schemaTestHasArgument(*lookups, name) {
					t.Errorf("missing %s preflight: %v", name, *lookups)
				}
			}
			const uri = "mongodb://localhost:27017/?appName=schema-test"
			const db = `production"; throw Error("injected");//`
			if err := plan.Deploy(uri, db); err != nil {
				t.Fatalf("Deploy: %v", err)
			}
			if len(*calls) != 2 || (*calls)[0].name != "mongosh" || (*calls)[1].name != "mongoimport" {
				t.Fatalf("expected schema command before import, got %+v", *calls)
			}
			shell, importer := (*calls)[0], (*calls)[1]
			if !schemaTestHasArgument(shell.args, "--quiet") || !schemaTestHasArgument(shell.args, "--norc") || !schemaTestHasArgument(shell.args, uri) {
				t.Fatalf("unexpected mongosh arguments: %v", shell.args)
			}
			script := schemaTestArgument(t, shell.args, "--eval")
			dbJSON, _ := json.Marshal(db)
			schemaJSON, _ := json.Marshal(schema)
			if !strings.Contains(script, "getSiblingDB("+string(dbJSON)+")") || !strings.Contains(script, "JSON.parse("+string(schemaJSON)+")") {
				t.Fatalf("script does not safely encode database and schema: %s", script)
			}
			drop := regexp.MustCompile(`runCommand\(\{\s*drop\s*:`).FindStringIndex(script)
			create := regexp.MustCompile(`runCommand\(\{\s*create\s*:`).FindStringIndex(script)
			if drop == nil || create == nil || create[0] <= drop[0] || !strings.Contains(script, "$jsonSchema") || !regexp.MustCompile(`\.code\s*!==\s*26`).MatchString(script) {
				t.Fatalf("script must drop then create the validated collection and handle absent namespaces: %s", script)
			}
			if len(regexp.MustCompile(`\.ok\s*!==\s*1`).FindAllString(script, -1)) != 2 {
				t.Fatalf("drop and create command results must both be checked: %s", script)
			}
			for _, pattern := range []string{`["']?validationLevel["']?\s*:\s*["']strict["']`, `["']?validationAction["']?\s*:\s*["']error["']`} {
				if !regexp.MustCompile(pattern).MatchString(script) {
					t.Errorf("script missing strict error validation: %s", script)
				}
			}
			if schemaTestArgument(t, importer.args, "--uri") != uri || schemaTestArgument(t, importer.args, "--db") != db || schemaTestArgument(t, importer.args, "--collection") != "blog_posts" {
				t.Fatalf("incorrect import target: %v", importer.args)
			}
			for _, flag := range []string{"--stopOnError", "--maintainInsertionOrder"} {
				if !schemaTestHasArgument(importer.args, flag) {
					t.Errorf("validated import missing %s: %v", flag, importer.args)
				}
			}
			for _, flag := range []string{"--file", "--drop", "--bypassDocumentValidation"} {
				if schemaTestHasArgument(importer.args, flag) {
					t.Errorf("validated import must not use %s: %v", flag, importer.args)
				}
			}
			if schemaTestHasArgument(importer.args, "--jsonArray") != source.jsonArray || string(importer.input) != source.wantInput {
				t.Fatalf("incorrect source payload: args=%v input=%q", importer.args, importer.input)
			}
		})
	}
}

func TestSchemaDeploymentStopsOnCommandFailure(t *testing.T) {
	for _, failingName := range []string{"mongosh", "mongoimport"} {
		t.Run(failingName, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			schemaTestWrite(t, root, "data/z-later.json", `{"later":true}`)
			schemaTestCommands(t)
			var commands []string
			failure := errors.New("command failed")
			runCommand = func(name string, _ []string, _ []byte) error {
				commands = append(commands, name)
				if name == failingName {
					return failure
				}
				return nil
			}
			if err := DeployAll("mongodb://localhost:27017", "test", dataDir, opts); !errors.Is(err, failure) {
				t.Fatalf("DeployAll should propagate command failure, got %v", err)
			}
			want := []string{"mongosh"}
			if failingName == "mongoimport" {
				want = append(want, "mongoimport")
			}
			if !reflect.DeepEqual(commands, want) {
				t.Fatalf("continued after failed %s: got %v, want %v", failingName, commands, want)
			}
		})
	}
}

func TestSchemaDeploymentRejectsInvalidSchema(t *testing.T) {
	for _, content := range []string{
		``, `{"bsonType":`, `[]`, `null`, `"object"`, `true`,
		`{"$jsonSchema":{"bsonType":"object"}}`,
		`{"validator":{"$jsonSchema":{"bsonType":"object"}}}`,
		`{"validationLevel":"strict"}`, `{"validationAction":"error"}`, `{"indexes":[]}`,
	} {
		t.Run(content, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			schemaTestWrite(t, root, "schemas/blog-posts.json", content)
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("expected invalid schema to fail preparation")
			}
			if len(*calls) != 0 {
				t.Fatalf("invalid schema ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentChecksSchemaNumberPrecisionBeforeMutation(t *testing.T) {
	for _, test := range []struct {
		number    string
		wantError bool
	}{
		{number: "9007199254740993", wantError: true},
		{number: "-9007199254740993", wantError: true},
		{number: "1e20", wantError: true},
		{number: "1e309", wantError: true},
		{number: "1e-400", wantError: true},
		{number: "9007199254740991"},
		{number: "-9007199254740991"},
		{number: "0.1"},
		{number: "0e-400"},
	} {
		t.Run(test.number, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			// Nest the value in both an object and an array so numeric checks must
			// inspect the complete schema rather than just its top-level fields.
			schema := `{"bsonType":"object","properties":{"value":{"enum":[` + test.number + `]}}}`
			schemaTestWrite(t, root, "schemas/blog-posts.json", schema)
			_, calls := schemaTestCommands(t)
			_, err := PrepareCollection("blog-posts", dataDir, opts)
			if (err != nil) != test.wantError {
				t.Fatalf("PrepareCollection schema number %s: got error %v, want error %v", test.number, err, test.wantError)
			}
			if len(*calls) != 0 {
				t.Fatalf("schema numeric preflight ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentRejectsUnreadableSchemaBeforeMutation(t *testing.T) {
	_, dataDir, opts := schemaTestFixture(t)
	// A directory reliably fails os.ReadFile even when tests run as a user that
	// can read permission-restricted files.
	opts.Schemas["blog-posts"] = "schemas"
	_, calls := schemaTestCommands(t)
	if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
		t.Fatal("schema directory must fail preparation")
	}
	if len(*calls) != 0 {
		t.Fatalf("unreadable schema ran a process: %+v", *calls)
	}
}

func TestSchemaDeploymentValidatesMappings(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*DeployOptions)
	}{
		{name: "mongo name instead of source key", edit: func(o *DeployOptions) { o.Schemas = map[string]string{"blog_posts": "schemas/blog-posts.json"} }},
		{name: "empty schema path", edit: func(o *DeployOptions) { o.Schemas["blog-posts"] = "" }},
		{name: "whitespace schema path", edit: func(o *DeployOptions) { o.Schemas["blog-posts"] = " \t " }},
		{name: "missing project root", edit: func(o *DeployOptions) { o.ProjectRoot = "" }},
		{name: "relative project root", edit: func(o *DeployOptions) { o.ProjectRoot = "." }},
		{name: "missing selected schema", edit: func(o *DeployOptions) { o.Schemas["blog-posts"] = "schemas/missing.json" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, dataDir, opts := schemaTestFixture(t)
			test.edit(&opts)
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("expected invalid schema mapping to fail")
			}
			if len(*calls) != 0 {
				t.Fatalf("invalid mapping ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentReadsOnlySelectedSchema(t *testing.T) {
	root, dataDir, opts := schemaTestFixture(t)
	schemaTestWrite(t, root, "data/other.json", `{"other":true}`)
	opts.Schemas["other"] = "schemas/does-not-exist.json"
	_, calls := schemaTestCommands(t)
	plan, err := PrepareCollection("blog-posts", dataDir, opts)
	if err != nil {
		t.Fatalf("unselected missing schema should not prevent preparation: %v", err)
	}
	if err := plan.Deploy("mongodb://localhost:27017", "test"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 {
		t.Fatalf("expected one configured collection deployment: %+v", *calls)
	}
}

func TestSchemaDeploymentPreflightsAllBeforeMutation(t *testing.T) {
	for _, failure := range []string{"schema", "mongosh", "mongoimport"} {
		t.Run(failure, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			schemaTestWrite(t, root, "data/a-legacy.json", `{"legacy":true}`)
			_, calls := schemaTestCommands(t)
			if failure == "schema" {
				schemaTestWrite(t, root, "schemas/blog-posts.json", `{"invalid":`)
			} else {
				lookPath = func(name string) (string, error) {
					if name == failure {
						return "", errors.New("not installed")
					}
					return name, nil
				}
			}
			if err := DeployAll("mongodb://localhost:27017", "test", dataDir, opts); err == nil {
				t.Fatalf("expected %s preflight failure", failure)
			}
			if len(*calls) != 0 {
				t.Fatalf("preflight failure changed a collection: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentRejectsNonObjectDocuments(t *testing.T) {
	for _, content := range []string{`[null]`, `[1]`, `["text"]`, `[[]]`, `[{"slug":"valid"},false]`} {
		t.Run(content, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			schemaTestWrite(t, root, "data/blog-posts.json", content)
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("configured source array must contain only JSON objects")
			}
			if len(*calls) != 0 {
				t.Fatalf("invalid source ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentRejectsInvalidUTF8(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		content string
	}{
		{name: "schema", path: "schemas/blog-posts.json", content: "{\"bsonType\":\"object\",\"description\":\"\xff\"}"},
		{name: "object source", path: "data/blog-posts.json", content: "{\"slug\":\"\xff\"}"},
		{name: "array source", path: "data/blog-posts.json", content: "[{\"slug\":\"\xff\"}]"},
		{name: "directory source", path: "data/blog-posts/01.json", content: "{\"slug\":\"\xff\"}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			if test.name == "directory source" {
				if err := os.Remove(filepath.Join(dataDir, "blog-posts.json")); err != nil {
					t.Fatal(err)
				}
			}
			schemaTestWrite(t, root, test.path, test.content)
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("configured schema and source must contain valid UTF-8")
			}
			if len(*calls) != 0 {
				t.Fatalf("invalid UTF-8 ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentLegacyUTF8HandlingUnchanged(t *testing.T) {
	root := t.TempDir()
	schemaTestWrite(t, root, "legacy.json", "{\"text\":\"\xff\"}")
	lookups, calls := schemaTestCommands(t)
	if err := DeployCollection("mongodb://localhost:27017", "test", "legacy", root); err != nil {
		t.Fatalf("legacy deployment added UTF-8 prevalidation: %v", err)
	}
	if len(*lookups) != 0 || len(*calls) != 1 || (*calls)[0].name != "mongoimport" || !schemaTestHasArgument((*calls)[0].args, "--file") {
		t.Fatalf("legacy import changed: lookups=%v commands=%+v", *lookups, *calls)
	}
}

func TestSchemaDeploymentRejectsSchemaInsideDataDirectory(t *testing.T) {
	for _, kind := range []string{"direct", "schema symlink", "data symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			inside := schemaTestWrite(t, root, "data/schema.json", `{"bsonType":"object"}`)
			switch kind {
			case "direct":
				opts.Schemas["blog-posts"] = "data/schema.json"
			case "schema symlink":
				if err := os.Symlink(inside, filepath.Join(root, "schemas", "linked.json")); err != nil {
					t.Fatal(err)
				}
				opts.Schemas["blog-posts"] = "schemas/linked.json"
			case "data symlink":
				linked := filepath.Join(root, "linked-data")
				if err := os.Symlink(dataDir, linked); err != nil {
					t.Fatal(err)
				}
				dataDir = linked
				opts.Schemas["blog-posts"] = "data/schema.json"
			}
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("schema in resolved data directory must be rejected")
			}
			if len(*calls) != 0 {
				t.Fatalf("unsafe schema path ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentRejectsSchemaAlsoDiscoveredAsSource(t *testing.T) {
	for _, source := range []string{"validator-copy.json", "other-collection/validator-copy.json"} {
		t.Run(source, func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			link := filepath.Join(dataDir, source)
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, "schemas", "blog-posts.json"), link); err != nil {
				t.Fatal(err)
			}
			_, calls := schemaTestCommands(t)
			if _, err := PrepareCollection("blog-posts", dataDir, opts); err == nil {
				t.Fatal("schema reachable through a discovered source must be rejected")
			}
			if len(*calls) != 0 {
				t.Fatalf("schema also used as source ran a process: %+v", *calls)
			}
		})
	}
}

func TestSchemaDeploymentUsesPreparedSnapshots(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[directory], func(t *testing.T) {
			root, dataDir, opts := schemaTestFixture(t)
			sourcePath := filepath.Join(dataDir, "blog-posts.json")
			want := `[{"slug":"first"}]`
			if directory {
				if err := os.Remove(sourcePath); err != nil {
					t.Fatal(err)
				}
				sourcePath = schemaTestWrite(t, dataDir, "blog-posts/01.json", `{"slug":"first"}`)
				want = "{\"slug\":\"first\"}\n"
			}
			_, calls := schemaTestCommands(t)
			plan, err := PrepareCollection("blog-posts", dataDir, opts)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(sourcePath); err != nil {
				t.Fatal(err)
			}
			schemaTestWrite(t, root, "schemas/blog-posts.json", `{"bsonType":"object","required":["changed"]}`)
			if err := plan.Deploy("mongodb://localhost:27017", "test"); err != nil {
				t.Fatal(err)
			}
			if len(*calls) != 2 || string((*calls)[1].input) != want {
				t.Fatalf("import did not use prepared source: %+v", *calls)
			}
			script := schemaTestArgument(t, (*calls)[0].args, "--eval")
			original, _ := json.Marshal(`{"bsonType":"object","required":["slug"]}`)
			if !strings.Contains(script, "JSON.parse("+string(original)+")") {
				t.Fatalf("schema changed after preparation: %s", script)
			}
		})
	}
}

func TestSchemaDeploymentUnselectedSchemaDoesNotChangeLegacyImport(t *testing.T) {
	root, dataDir, opts := schemaTestFixture(t)
	opts.Schemas["blog-posts"] = "schemas/missing.json"
	schemaTestWrite(t, root, "data/legacy.json", `{"legacy":true}`)
	lookups, calls := schemaTestCommands(t)
	if err := DeployCollection("mongodb://localhost:27017", "test", "legacy", dataDir, opts); err != nil {
		t.Fatalf("unselected schema changed legacy import: %v", err)
	}
	if len(*lookups) != 0 || len(*calls) != 1 || (*calls)[0].name != "mongoimport" || !schemaTestHasArgument((*calls)[0].args, "--drop") {
		t.Fatalf("legacy deployment used schema path: lookups=%v commands=%+v", *lookups, *calls)
	}
}

func TestSchemaDeploymentLegacyImportUnchanged(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[directory], func(t *testing.T) {
			root := t.TempDir()
			dataDir := filepath.Join(root, "data")
			file := "news.json"
			content := `[{"title":"first"}]`
			if directory {
				file, content = "news/01.json", `{"title":"first"}`
			}
			path := schemaTestWrite(t, dataDir, file, content)
			lookups, calls := schemaTestCommands(t)
			if err := DeployCollection("mongodb://localhost:27017", "test", "news", dataDir); err != nil {
				t.Fatal(err)
			}
			if len(*lookups) != 0 || len(*calls) != 1 || (*calls)[0].name != "mongoimport" {
				t.Fatalf("legacy deployment changed tool requirements: lookups=%v commands=%+v", *lookups, *calls)
			}
			wantArgs := []string{"--uri", "mongodb://localhost:27017", "--db", "test", "--collection", "news"}
			var wantInput []byte
			if directory {
				wantInput = []byte(content + "\n")
			} else {
				wantArgs = append(wantArgs, "--file", path)
			}
			wantArgs = append(wantArgs, "--drop")
			if !directory {
				wantArgs = append(wantArgs, "--jsonArray")
			}
			if !reflect.DeepEqual((*calls)[0].args, wantArgs) || !reflect.DeepEqual((*calls)[0].input, wantInput) {
				t.Fatalf("legacy invocation changed: got %+v, want args=%v input=%q", (*calls)[0], wantArgs, wantInput)
			}
		})
	}
}
