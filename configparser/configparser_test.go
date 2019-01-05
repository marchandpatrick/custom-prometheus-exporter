package configparser

import (
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/assert"
)

func createFile(t *testing.T, data string) string {
	filename := "/tmp/customPromExporterTest.data"
	assert.NilError(t, ioutil.WriteFile(filename, []byte(data), 0644))
	return filename
}

func removeFile(name string) {
	os.Remove(name)
}

func TestSingleConfigFile(t *testing.T) {
	c := Config{ConfigFiles: []string{"../example-configurations/test-exporter.yaml"}}
	assert.NilError(t, c.ParseConfig())
}

func TestTwoConfigFiles(t *testing.T) {
	c := Config{ConfigFiles: []string{"../example-configurations/test-exporter.yaml", "../example-configurations/docker-exporter.yaml"}}
	assert.NilError(t, c.ParseConfig())
}

func TestMissingSingleConfigFile(t *testing.T) {
	c := Config{ConfigFiles: []string{"missing.yaml"}}
	assert.ErrorContains(t, c.ParseConfig(), "missing.yaml: no such file or directory")
}

func TestMissingFirstConfigFile(t *testing.T) {
	c := Config{ConfigFiles: []string{"missing.yaml", "../example-configurations/docker-exporter.yaml"}}
	assert.ErrorContains(t, c.ParseConfig(), "missing.yaml: no such file or directory")
}

func TestMissingSecondConfigFile(t *testing.T) {
	c := Config{ConfigFiles: []string{"../example-configurations/test-exporter.yaml", "missing.yaml"}}
	assert.ErrorContains(t, c.ParseConfig(), "missing.yaml: no such file or directory")

}
func TestInvalidConfigFile(t *testing.T) {
	errorTag := "unexpectedTag"

	data := `
name: test-exporter
port: 12345
endpoint: /test
` + errorTag + `: true    # This is not valid and should cause a parse failure
metrics:
- name: test_gauge_values
  help: Some values
  type: gauge
  executions:
  - type: sh
    command: expr 111
    labels:
      order: first
`
	filename := createFile(t, data)
	defer removeFile(filename)

	c := Config{ConfigFiles: []string{filename}}
	assert.ErrorContains(t, c.ParseConfig(), "field "+errorTag+" not found")
}

func TestMissingName(t *testing.T) {
	data := `
#name: test-exporter       Missing field should cause error
port: 12345
endpoint: /test
metrics:
- name: test_gauge_values
  help: Some values
  type: gauge
  executions:
  - type: sh
    command: expr 111
    labels:
      order: first
`

	filename := createFile(t, data)
	defer removeFile(filename)

	c := Config{ConfigFiles: []string{filename}}
	assert.ErrorContains(t, c.ParseConfig(), "field name not found")
}

func TestMissingPort(t *testing.T) {
	data := `
name: test-exporter
#port: 12345          Missing field should cause error
endpoint: /test
metrics:
- name: test_gauge_values
  help: Some values
  type: gauge
  executions:
  - type: sh
    command: expr 111
    labels:
      order: first
`

	filename := createFile(t, data)
	defer removeFile(filename)

	c := Config{ConfigFiles: []string{filename}}
	assert.ErrorContains(t, c.ParseConfig(), "field port not found")
}

func TestMissingEndpoint(t *testing.T) {
	data := `
name: test-exporter
port: 12345
#endpoint: /test         Missing field should default to /metrics
metrics:
- name: test_gauge_values
  help: Some values
  type: gauge
  executions:
  - type: sh
    command: expr 111
    labels:
      order: first
`
	filename := createFile(t, data)
	defer removeFile(filename)

	c := Config{ConfigFiles: []string{filename}}
	assert.NilError(t, c.ParseConfig())
	assert.Equal(t, c.Exporters[0].Endpoint, "/metrics")
}

func TestMissingMetrics(t *testing.T) {
	data := `
name: test-exporter
port: 12345
endpoint: /test
#metrics:             Missing field should cause error
`
	filename := createFile(t, data)
	defer removeFile(filename)

	c := Config{ConfigFiles: []string{filename}}
	assert.ErrorContains(t, c.ParseConfig(), "field metrics not found")

}