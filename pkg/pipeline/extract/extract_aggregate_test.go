/*
 * Copyright (C) 2022 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package extract

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	"github.com/netobserv/flowlogs-pipeline/pkg/pipeline/extract/aggregate"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

// This tests extract_aggregate as a whole. It can be thought of as an integration test between extract_aggregate.go and
// aggregate.go and aggregates.go. The test sends flows in 2 batches and verifies the extractor's output after each
// batch. The output of the 2nd batch depends on the 1st batch.
func Test_Extract(t *testing.T) {
	// Setup
	yamlConfig := `
aggregates:
- name: bandwidth
  by:
  - service
  operation: sum
  recordkey: bytes
- name: bandwidth_count
  by:
  - service
  operation: count
  recordkey: ""
`
	var err error
	yamlData := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(yamlConfig), &yamlData)
	require.NoError(t, err)
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	jsonBytes, err := json.Marshal(yamlData["aggregates"])
	require.NoError(t, err)
	config.Opt.PipeLine.Extract.Aggregates = string(jsonBytes)

	extractAggregate, err := NewExtractAggregate()
	require.NoError(t, err)

	// Test cases
	tests := []struct {
		name         string
		inputBatch   []config.GenericMap
		expectedAggs []config.GenericMap
	}{
		{
			name: "batch1",
			inputBatch: []config.GenericMap{
				{"service": "http", "bytes": 10.0},
				{"service": "http", "bytes": 20.0},
				{"service": "tcp", "bytes": 1.0},
				{"service": "tcp", "bytes": 2.0},
			},
			expectedAggs: []config.GenericMap{
				{
					"name":            "bandwidth",
					"record_key":      "bytes",
					"by":              "service",
					"aggregate":       "http",
					"service":         "http",
					"operation":       aggregate.Operation(aggregate.OperationSum),
					"value":           "30.000000",
					"bandwidth_value": "30.000000",
					"recentRawValues": []float64{10.0, 20.0},
					"count":           "2",
				},
				{
					"name":            "bandwidth",
					"record_key":      "bytes",
					"by":              "service",
					"aggregate":       "tcp",
					"service":         "tcp",
					"operation":       aggregate.Operation(aggregate.OperationSum),
					"value":           "3.000000",
					"bandwidth_value": "3.000000",
					"recentRawValues": []float64{1.0, 2.0},
					"count":           "2",
				},
				{
					"name":                  "bandwidth_count",
					"record_key":            "",
					"by":                    "service",
					"aggregate":             "http",
					"service":               "http",
					"operation":             aggregate.Operation(aggregate.OperationCount),
					"value":                 "2.000000",
					"bandwidth_count_value": "2.000000",
					"recentRawValues":       []float64{1.0, 1.0},
					"count":                 "2",
				},
				{
					"name":                  "bandwidth_count",
					"record_key":            "",
					"by":                    "service",
					"aggregate":             "tcp",
					"service":               "tcp",
					"operation":             aggregate.Operation(aggregate.OperationCount),
					"value":                 "2.000000",
					"bandwidth_count_value": "2.000000",
					"recentRawValues":       []float64{1.0, 1.0},
					"count":                 "2",
				},
			},
		},
		{
			name: "batch2",
			inputBatch: []config.GenericMap{
				{"service": "http", "bytes": 30.0},
				{"service": "tcp", "bytes": 4.0},
				{"service": "tcp", "bytes": 5.0},
			},
			expectedAggs: []config.GenericMap{
				{
					"name":            "bandwidth",
					"record_key":      "bytes",
					"by":              "service",
					"aggregate":       "http",
					"service":         "http",
					"operation":       aggregate.Operation(aggregate.OperationSum),
					"value":           "60.000000",
					"bandwidth_value": "60.000000",
					"recentRawValues": []float64{30.0},
					"count":           "3",
				},
				{
					"name":            "bandwidth",
					"record_key":      "bytes",
					"by":              "service",
					"aggregate":       "tcp",
					"service":         "tcp",
					"operation":       aggregate.Operation(aggregate.OperationSum),
					"value":           "12.000000",
					"bandwidth_value": "12.000000",
					"recentRawValues": []float64{4.0, 5.0},
					"count":           "4",
				},
				{
					"name":                  "bandwidth_count",
					"record_key":            "",
					"by":                    "service",
					"aggregate":             "http",
					"service":               "http",
					"operation":             aggregate.Operation(aggregate.OperationCount),
					"value":                 "3.000000",
					"bandwidth_count_value": "3.000000",
					"recentRawValues":       []float64{1.0},
					"count":                 "3",
				},
				{
					"name":                  "bandwidth_count",
					"record_key":            "",
					"by":                    "service",
					"aggregate":             "tcp",
					"service":               "tcp",
					"operation":             aggregate.Operation(aggregate.OperationCount),
					"value":                 "4.000000",
					"bandwidth_count_value": "4.000000",
					"recentRawValues":       []float64{1.0, 1.0},
					"count":                 "4",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualAggs := extractAggregate.Extract(tt.inputBatch)
			// Since the order of the elements in the returned slice from Extract() is non-deterministic, we use
			// ElementsMatch() rather than Equals()
			require.ElementsMatch(t, tt.expectedAggs, actualAggs)
		})
	}
}
