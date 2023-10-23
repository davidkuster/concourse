package atc

import (
	"bytes"
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigFormatting1(t *testing.T) {
	p := `
# groups!
groups:

  - jobs:
      - pull-request-job
    name: pull-request

# resource types!
resource_types:

  - type: registry-image
    source:
      repository: teliaoss/github-pr-resource
    name: pull-request

# resources!
resources:

  - name: pull-request
    type: pull-request
    check_every: 24h0m0s
    webhook_token: ((service-account.webhook))
    source:
      access_token: ((service-account.access-token))
      repository: github-org/my-repo
      v3_endpoint: https://api.github.com
      v4_endpoint: https://api.github.com/graphql

# jobs!
jobs:

  - name: pull-request-job
    plan:
      - get: pull-request
        trigger: true
        version: every
      - in_parallel:
          steps:
            - file: pull-request/ci/tasks/base/task_lint_go.yaml
              input_mapping:
                code: pull-request
              task: pull-request-lint
`

	var config Config
	if err := UnmarshalConfig([]byte(p), &config); err != nil {
		t.Fatalf("error unmarshaling yaml %s: %v", p, err)
	}

	node := yaml.Node{}
	if err := node.Encode(&config); err != nil {
		t.Errorf("error encoding yaml: %v", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		t.Errorf("error encoding node: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Errorf("error closing encoder: %v", err)
	}

	fmt.Printf("output = %s", buf.String())
}

func TestConfigFormatting2(t *testing.T) {
	p := `
# jobs!
jobs:

        # mis-aligned head comment
  - build_logs_to_retain: 10
    ensure:
      put: resource-to-always-put
    name: job-1
    on_failure:
      put: resource-to-put-on-failure
       # mis-aligned tail comment on put
    plan:
      - file: /path/to/script.sh
        privileged: true
        task: task-1
      - get: resource-to-get
        trigger: true
       # mis-aligned tail comment on get
    serial: true
`

	var config Config
	if err := UnmarshalConfig([]byte(p), &config); err != nil {
		t.Fatalf("error unmarshaling yaml %s: %v", p, err)
	}

	node := yaml.Node{}
	if err := node.Encode(&config); err != nil {
		t.Errorf("error encoding yaml: %v", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		t.Errorf("error encoding node: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Errorf("error closing encoder: %v", err)
	}

	fmt.Printf("output = %s", buf.String())
}
