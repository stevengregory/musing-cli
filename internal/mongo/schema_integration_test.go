package mongo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stevengregory/musing-cli/internal/mongo"
)

func validateLocalMongoTestURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "mongodb" || parsed.Opaque != "" {
		return fmt.Errorf("test URI must use mongodb:// with an explicit local host and port")
	}
	if strings.TrimSpace(raw) != raw || parsed.User != nil || parsed.Path != "" || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || strings.ContainsAny(raw, "?#") {
		return fmt.Errorf("test URI must not contain credentials, a database path, a query, or a fragment")
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return fmt.Errorf("test URI must address only 127.0.0.1, localhost, or [::1] with an explicit port")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("test URI requires a port between 1 and 65535")
	}
	return nil
}

func TestValidateLocalMongoTestURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{name: "IPv4", uri: "mongodb://127.0.0.1:27017", want: true},
		{name: "localhost", uri: "mongodb://localhost:27018", want: true},
		{name: "IPv6", uri: "mongodb://[::1]:27019", want: true},
		{name: "unset"},
		{name: "remote", uri: "mongodb://db.example.com:27017"},
		{name: "all interfaces", uri: "mongodb://0.0.0.0:27017"},
		{name: "SRV", uri: "mongodb+srv://localhost:27017"},
		{name: "missing port", uri: "mongodb://localhost"},
		{name: "empty port", uri: "mongodb://localhost:"},
		{name: "zero port", uri: "mongodb://localhost:0"},
		{name: "large port", uri: "mongodb://localhost:65536"},
		{name: "credentials", uri: "mongodb://test:secret@localhost:27017"},
		{name: "database", uri: "mongodb://localhost:27017/existing"},
		{name: "root path", uri: "mongodb://localhost:27017/"},
		{name: "query", uri: "mongodb://localhost:27017?directConnection=true"},
		{name: "empty query", uri: "mongodb://localhost:27017?"},
		{name: "fragment", uri: "mongodb://localhost:27017#test"},
		{name: "empty fragment", uri: "mongodb://localhost:27017#"},
		{name: "multiple hosts", uri: "mongodb://localhost:27017,db.example.com:27017"},
		{name: "whitespace", uri: " mongodb://localhost:27017"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateLocalMongoTestURI(tt.uri) == nil; got != tt.want {
				t.Fatalf("URI accepted = %t, want %t", got, tt.want)
			}
		})
	}
}

// Opting in acknowledges that the supplied local endpoint is disposable. This
// test creates and drops only its uniquely named database, never an existing one.
func TestMongoSchemaValidatorIntegration(t *testing.T) {
	uri := os.Getenv("MUSING_MONGO_TEST_URI")
	if os.Getenv("MUSING_MONGO_INTEGRATION") != "1" || uri == "" {
		t.Skip("requires MUSING_MONGO_INTEGRATION=1 and an explicit MUSING_MONGO_TEST_URI for a disposable local MongoDB")
	}
	if err := validateLocalMongoTestURI(uri); err != nil {
		t.Fatal(err)
	}
	shell, err := exec.LookPath("mongosh")
	if err != nil {
		t.Fatal("opted-in integration test requires mongosh")
	}
	if _, err := exec.LookPath("mongoimport"); err != nil {
		t.Fatal("opted-in integration test requires mongoimport")
	}

	dbName := "musing_schema_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	dbJSON := mongoTestJSON(t, dbName)
	t.Cleanup(func() {
		output, err := runMongoTestScript(shell, uri, "const result = db.getSiblingDB("+dbJSON+").dropDatabase(); if (result.ok !== 1) throw new Error('test database cleanup failed');")
		if err != nil {
			t.Errorf("test database cleanup failed: %s\n%s", redactMongoTestURI(err.Error(), uri), output)
		}
	})

	projectRoot := t.TempDir()
	dataDir := filepath.Join(projectRoot, "data")
	schemaDir := filepath.Join(projectRoot, "schemas")
	for _, dir := range []string{dataDir, schemaDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dataPath := filepath.Join(dataDir, "test-records.json")
	schemaPath := filepath.Join(schemaDir, "test-records.json")
	options := mongo.DeployOptions{
		ProjectRoot: projectRoot,
		Schemas: map[string]string{
			"test-records": filepath.Join("schemas", "test-records.json"),
		},
	}
	firstSchema := json.RawMessage(`{"bsonType":"object","required":["label"],"properties":{"label":{"bsonType":"string","enum":["first"]}}}`)
	secondSchema := json.RawMessage(`{"bsonType":"object","required":["label","stage"],"properties":{"label":{"bsonType":"string","enum":["second"]},"stage":{"bsonType":"string","enum":["ready"]}}}`)

	writeMongoTestFixture(t, schemaPath, firstSchema)
	writeMongoTestFixture(t, dataPath, []byte(`{"label":"first"}`))
	if output, err := deployMongoTestCollection(t, uri, dbName, dataDir, options); err != nil {
		t.Fatalf("valid document deployment failed: %s\n%s", redactMongoTestURI(err.Error(), uri), output)
	}
	assertMongoTestState(t, shell, uri, dbName, firstSchema, []map[string]any{{"label": "first"}})

	// Changing both the required fields and allowed label verifies that the old
	// collection and validator are replaced together before importing new data.
	writeMongoTestFixture(t, schemaPath, secondSchema)
	writeMongoTestFixture(t, dataPath, []byte(`{"label":"second","stage":"ready"}`))
	if output, err := deployMongoTestCollection(t, uri, dbName, dataDir, options); err != nil {
		t.Fatalf("replacement deployment failed: %s\n%s", redactMongoTestURI(err.Error(), uri), output)
	}
	assertMongoTestState(t, shell, uri, dbName, secondSchema, []map[string]any{{"label": "second", "stage": "ready"}})

	writeMongoTestFixture(t, dataPath, []byte(`{"label":17,"stage":"ready"}`))
	if _, err := deployMongoTestCollection(t, uri, dbName, dataDir, options); err == nil {
		t.Fatal("deployment accepted a document that violates the installed schema")
	}
	// A failed import need not preserve the previous snapshot atomically, but
	// its validator must remain installed and the invalid record must be absent.
	assertMongoTestState(t, shell, uri, dbName, secondSchema, nil)
	script := "const collection = db.getSiblingDB(" + dbJSON + ").getCollection('test_records'); if (collection.countDocuments({label:17}) !== 0) throw new Error('invalid record was stored');"
	if output, err := runMongoTestScript(shell, uri, script); err != nil {
		t.Fatalf("invalid record assertion failed: %s\n%s", redactMongoTestURI(err.Error(), uri), output)
	}
}

func writeMongoTestFixture(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mongoTestJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func redactMongoTestURI(output, uri string) string {
	return strings.ReplaceAll(output, uri, "[local test endpoint]")
}

func runMongoTestScript(shell, uri, script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, shell, "--quiet", "--norc", uri, "--eval", script).CombinedOutput()
	return redactMongoTestURI(string(output), uri), err
}

func deployMongoTestCollection(t *testing.T, uri, dbName, dataDir string, options mongo.DeployOptions) (string, error) {
	t.Helper()
	outputPath := filepath.Join(t.TempDir(), "deployment-output")
	output, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	// DeployCollection forwards tool output to process streams. Capture it so
	// test failures can show diagnostics without printing the supplied URI.
	err = func() error {
		stdout, stderr := os.Stdout, os.Stderr
		defer func() { os.Stdout, os.Stderr = stdout, stderr }()
		os.Stdout, os.Stderr = output, output
		return mongo.DeployCollection(uri, dbName, "test-records", dataDir, options)
	}()
	if closeErr := output.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	captured, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return redactMongoTestURI(string(captured), uri), err
}

func assertMongoTestState(t *testing.T, shell, uri, dbName string, schema json.RawMessage, documents []map[string]any) {
	t.Helper()
	script := `
const testDB = db.getSiblingDB(` + mongoTestJSON(t, dbName) + `);
const info = testDB.getCollectionInfos({name: "test_records"})[0];
if (!info) throw new Error("collection is missing");
if (info.options.validationLevel !== "strict") throw new Error("validation level is not strict");
if (info.options.validationAction !== "error") throw new Error("validation action is not error");
function canonical(value) {
    if (Array.isArray(value)) return value.map(canonical);
    if (value && typeof value === "object") {
        return Object.fromEntries(Object.keys(value).sort().map(key => [key, canonical(value[key])]));
    }
    return value;
}
const expectedSchema = ` + mongoTestJSON(t, schema) + `;
if (JSON.stringify(canonical(info.options.validator)) !== JSON.stringify(canonical({$jsonSchema: expectedSchema}))) {
    throw new Error("installed validator does not match the configured schema");
}
const expectedDocuments = ` + mongoTestJSON(t, documents) + `;
if (expectedDocuments !== null) {
    const actualDocuments = testDB.getCollection("test_records").find({}, {_id: 0}).sort({label: 1}).toArray();
    if (JSON.stringify(canonical(actualDocuments)) !== JSON.stringify(canonical(expectedDocuments))) {
        throw new Error("collection documents do not match the replacement snapshot");
    }
}
`
	if output, err := runMongoTestScript(shell, uri, script); err != nil {
		t.Fatalf("collection state assertion failed: %s\n%s", redactMongoTestURI(err.Error(), uri), output)
	}
}
