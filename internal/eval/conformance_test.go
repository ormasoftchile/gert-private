package eval

import (
"fmt"
"os"
"path/filepath"
"sort"
"strings"
"testing"

"github.com/ormasoftchile/gert/internal/eval/core"
"github.com/ormasoftchile/gert/internal/eval/gxl"
"gopkg.in/yaml.v3"
)

// Vector is one normalized conformance vector ready for dispatch.
type Vector struct {
ID                 string
Category           string
Description        string
Input              string
Variables          core.Value
ExpectedValue      core.Value
ExpectedHasValue   bool
ExpectedErrorClass string
ExpectedErrorCode  string
SourceFile         string
}

type vectorRunner func(Vector) (got core.Value, gotErrCode string, err error)

type conformanceStats struct {
Total int
Pass  int
Fail  int
Skip  int
ByCat map[string]*conformanceStats
}

type notImplementedError struct {
runner string
}

func (e notImplementedError) Error() string {
return fmt.Sprintf("not implemented: %s runner", e.runner)
}

func TestConformance(t *testing.T) {
files, err := filepath.Glob(filepath.Join("..", "..", "design", "gert", "conformance", "tv-*.yaml"))
if err != nil {
t.Fatalf("discover conformance corpus: %v", err)
}
sort.Strings(files)
if len(files) == 0 {
t.Fatal("discover conformance corpus: no tv-*.yaml files found")
}

stats := conformanceStats{ByCat: make(map[string]*conformanceStats)}
for _, file := range files {
vectors, err := loadCorpusVectors(file)
if err != nil {
t.Fatalf("load %s: %v", file, err)
}
runnerName, runner, err := runnerForFile(filepath.Base(file))
if err != nil {
t.Fatalf("dispatch %s: %v", file, err)
}
categoryStats := stats.category(runnerName)
for _, vector := range vectors {
status, detail := runConformanceVector(vector, runner)
stats.record(status)
categoryStats.record(status)
if testing.Verbose() {
t.Logf("%s %s %s: %s", runnerName, vector.ID, strings.ToUpper(status), detail)
}
}
}

fmt.Print(stats.summary())
if stats.Fail > 0 {
t.Fatalf("GERT conformance failed: %d/%d vectors failed", stats.Fail, stats.Total)
}
}

func (s *conformanceStats) category(name string) *conformanceStats {
if s.ByCat[name] == nil {
s.ByCat[name] = &conformanceStats{ByCat: make(map[string]*conformanceStats)}
}
return s.ByCat[name]
}

func (s *conformanceStats) record(status string) {
s.Total++
switch status {
case "pass":
s.Pass++
case "fail":
s.Fail++
case "skip":
s.Skip++
}
}

func (s conformanceStats) summary() string {
var builder strings.Builder
fmt.Fprintf(&builder, "GERT conformance summary: total=%d pass=%d fail=%d skip=%d\n", s.Total, s.Pass, s.Fail, s.Skip)
keys := make([]string, 0, len(s.ByCat))
for key := range s.ByCat {
keys = append(keys, key)
}
sort.Strings(keys)
for _, key := range keys {
cat := s.ByCat[key]
fmt.Fprintf(&builder, "  %s: total=%d pass=%d fail=%d skip=%d\n", key, cat.Total, cat.Pass, cat.Fail, cat.Skip)
}
return builder.String()
}

func runConformanceVector(vector Vector, runner vectorRunner) (string, string) {
got, gotErrCode, err := runner(vector)
if err != nil {
return "fail", err.Error()
}
if vector.ExpectedErrorCode != "" {
if gotErrCode == vector.ExpectedErrorCode {
return "pass", gotErrCode
}
return "fail", fmt.Sprintf("got error code %q, want %q", gotErrCode, vector.ExpectedErrorCode)
}
if vector.ExpectedHasValue {
if got.Equal(vector.ExpectedValue) {
return "pass", got.String()
}
return "fail", fmt.Sprintf("got %s, want %s", got, vector.ExpectedValue)
}
return "skip", "vector has no expected value or error code"
}

func runnerForFile(base string) (string, vectorRunner, error) {
switch base {
case "tv-gxl-parse.yaml":
return "GXL-PARSE", gxlParseRunner, nil
case "tv-gxl-eval.yaml":
return "GXL-EVAL", gxlEvalRunner, nil
case "tv-gxl-path.yaml":
return "GXL-PATH", gxlPathRunner, nil
case "tv-gis-path.yaml":
return "GIS-PATH", gisPathRunner, nil
case "tv-gcp-path.yaml":
return "GCP-PATH", gcpPathRunner, nil
default:
return "", nil, fmt.Errorf("no conformance runner for %s", base)
}
}

func gxlParseRunner(vector Vector) (core.Value, string, error) {
_, err := gxl.Parse(vector.Input)
if err != nil {
if parseErr, ok := err.(*gxl.ParseError); ok {
return core.Value{}, parseErr.Code, nil
}
return core.Value{}, "", err
}
return core.NewString("parse_ok"), "", nil
}

func gxlEvalRunner(Vector) (core.Value, string, error) {
return core.Value{}, "NOT-IMPLEMENTED", notImplementedError{runner: "GXL eval"}
}

func gxlPathRunner(Vector) (core.Value, string, error) {
return core.Value{}, "NOT-IMPLEMENTED", notImplementedError{runner: "GXL path"}
}

func gisPathRunner(Vector) (core.Value, string, error) {
return core.Value{}, "NOT-IMPLEMENTED", notImplementedError{runner: "GIS path"}
}

func gcpPathRunner(Vector) (core.Value, string, error) {
return core.Value{}, "NOT-IMPLEMENTED", notImplementedError{runner: "GCP path"}
}

func loadCorpusVectors(path string) ([]Vector, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, err
}

var document yaml.Node
if err := yaml.Unmarshal(data, &document); err != nil {
return nil, err
}
root := document.Content[0]
vectorsNode := mappingValue(root, "vectors")
if vectorsNode == nil || vectorsNode.Kind != yaml.SequenceNode {
return nil, fmt.Errorf("missing vectors sequence")
}

vectors := make([]Vector, 0, len(vectorsNode.Content))
for i, node := range vectorsNode.Content {
vector, err := parseVectorNode(filepath.Base(path), node)
if err != nil {
return nil, fmt.Errorf("vector %d: %w", i, err)
}
vectors = append(vectors, vector)
}
if len(vectors) == 0 {
return nil, fmt.Errorf("no vectors")
}
return vectors, nil
}

func parseVectorNode(sourceFile string, node *yaml.Node) (Vector, error) {
if node.Kind != yaml.MappingNode {
return Vector{}, fmt.Errorf("vector is not a mapping")
}
variablesNode := mappingValue(node, "variables")
variables := core.NewNull()
if variablesNode != nil {
value, err := core.FromYAML(variablesNode)
if err != nil {
return Vector{}, fmt.Errorf("variables: %w", err)
}
variables = value
}

vector := Vector{
ID:          mappingString(node, "id"),
Category:    mappingString(node, "category"),
Description: mappingString(node, "description"),
Input:       mappingString(node, "input"),
Variables:   variables,
SourceFile:  sourceFile,
}
if vector.ID == "" {
return Vector{}, fmt.Errorf("missing id")
}
if vector.Category == "" {
return Vector{}, fmt.Errorf("%s: missing category", vector.ID)
}

expectedNode := mappingValue(node, "expected")
if expectedNode == nil || expectedNode.Kind != yaml.MappingNode {
return Vector{}, fmt.Errorf("%s: missing expected mapping", vector.ID)
}
if valueNode := mappingValue(expectedNode, "value"); valueNode != nil {
value, err := core.FromYAML(valueNode)
if err != nil {
return Vector{}, fmt.Errorf("%s expected.value: %w", vector.ID, err)
}
vector.ExpectedValue = value
vector.ExpectedHasValue = true
}
vector.ExpectedErrorClass = mappingString(expectedNode, "error_class")
vector.ExpectedErrorCode = mappingString(expectedNode, "error_code")
return vector, nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
if node == nil || node.Kind != yaml.MappingNode {
return nil
}
for i := 0; i < len(node.Content); i += 2 {
if node.Content[i].Value == key {
return node.Content[i+1]
}
}
return nil
}

func mappingString(node *yaml.Node, key string) string {
value := mappingValue(node, key)
if value == nil {
return ""
}
return value.Value
}
