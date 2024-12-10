package prometheus

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinted/graphql-exporter/internal/config"
)

func TestValidateLabelKeysPath(t *testing.T) {
	tests := []struct {
		description string
		valueKey    string
		labelKeys   []config.Label
		expectErr   bool
	}{
		{
			description: "Valid case: a valueKey path can have one to n `*` which is not in the labelKeys",
			valueKey:    "l1.l2a.*.label2",
			labelKeys:   []config.Label{{Alias: "label1", Path: "l1.l2b.label1"}},
			expectErr:   false,
		},
		{
			description: "Case fail: a labelKey cannot have an * that is not in the valueKeys",
			valueKey:    "l1.l2a.label2",
			labelKeys:   []config.Label{{Alias: "label1", Path: "l1.l2b.*.label1"}},
			expectErr:   true,
		},
		{
			description: "Valid case: The labelKey can correspond to valueKey with *.",
			valueKey:    "l1.l2a.*.label1",
			labelKeys:   []config.Label{{Alias: "label1", Path: "l1.l2b.label1"}},
			expectErr:   false,
		},
		{
			description: "Valid case: LabelKey and valueKey match perfectly",
			valueKey:    "l1.l2a.label1",
			labelKeys:   []config.Label{{Alias: "label1", Path: "l1.l2a.label1"}},
			expectErr:   false,
		},
		{
			description: "Valid case: * in labelKey corresponds to any segment in valueKey",
			valueKey:    "l1.*.label1",
			labelKeys:   []config.Label{{Alias: "label1", Path: "l1.l2a.label1"}},
			expectErr:   false,
		},
		{
			description: "Case with an error : * in labelKey at wrong position",
			valueKey:    "l1.l2b.*.l3a.label2",
			labelKeys:   []config.Label{{Alias: "label2", Path: "l1.l2b.l3a.*.label2"}},
			expectErr:   true,
		},
		{
			description: "Valid case: labelKey has more segments than valueKey",
			valueKey:    "l1.l2a.*.label1",
			labelKeys:   []config.Label{{Alias: "moreLabel", Path: "l1.l2a.*.label1.moreLabel"}},
			expectErr:   false,
		},
		{
			description: "Valid case: valueKey and labelKey with * to match",
			valueKey:    "l1.l2b.*.l3a.label2",
			labelKeys:   []config.Label{{Alias: "label2", Path: "l1.l2b.*.l3a.label2"}},
			expectErr:   false,
		},
		{
			description: "Case with an error: * too much in a labelKey",
			valueKey:    "l1.l2b.*.l3a.label2",
			labelKeys:   []config.Label{{Alias: "label2", Path: "l1.l2b.*.l3a.*.label2"}},
			expectErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("valueKey: %s, labelKeys: %v", test.valueKey, test.labelKeys), func(t *testing.T) {
			// Call the validateLabelKeysPath function
			_, err := NewExtractor(".", test.valueKey, test.labelKeys)

			if (err != nil) != test.expectErr {
				t.Errorf("Error on test %s expected: %v, but got: %v", test.description, test.expectErr, err != nil)
			}
		})
	}
}

// Test-specific callback function to validate that each `expected` element has been validated at least once
func callbackTester(value string, labels []string, expected []ExpectedMetric, validated map[int]bool, t *testing.T) {
	count := 0
	for i, exp := range expected {
		// If the value and labels correspond to an expected element
		if value == exp.Metric && reflect.DeepEqual(labels, exp.Labels) {
			// Mark this item as validated
			validated[i] = true
			count++
			return
		}
	}
	if count != len(expected) {
		t.Errorf("Too many callback detected; expected %d, found %d", len(expected), count)
	}

	// If no match is found, log an error
	t.Errorf("Value '%s' with unexpected '%v' labels. No matching element in expectations: %v", value, labels, expected)
}

type ExpectedMetric struct {
	Metric string
	Labels []string
}

// Fonction de test
func TestExtractor(t *testing.T) {
	// JSON dataset
	data := map[string]interface{}{
		"l1": map[string]interface{}{
			"l2a": map[string]interface{}{
				"label1": "lab1val1",
			},
			"l2b": []interface{}{
				map[string]interface{}{
					"l3a": map[string]interface{}{
						"label2": "lab2val1",
						"label3": "lab3val1",
						"l4a": []interface{}{
							map[string]interface{}{
								"label4": "lab4val1",
								"label5": "1",
								"l5a": []interface{}{
									map[string]interface{}{
										"label6": "lab6val1",
									},
								},
							},
						},
					},
				},
				map[string]interface{}{
					"l3a": map[string]interface{}{
						"label2": "lab2val2",
						"label3": "lab3val2",
						"l4a": []interface{}{
							map[string]interface{}{
								"label4": "lab4val2",
								"label5": "2",
								"l5a": []interface{}{
									map[string]interface{}{
										"label6": "lab6val2",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Testing different paths and expected values
	tests := []struct {
		description string
		metricKey   string
		labelKeys   []config.Label
		expected    []ExpectedMetric
	}{
		{
			description: "Case 1: value with 1 array level only",
			metricKey:   "l1.l2b.*.l3a.label2",
			labelKeys: []config.Label{
				{Alias: "", Path: "l1.l2a.label1"},
				{Alias: "", Path: "l1.l2b.*.l3a.label3"},
			},
			expected: []ExpectedMetric{
				{Metric: "lab2val1", Labels: []string{"lab1val1", "lab3val1"}},
				{Metric: "lab2val2", Labels: []string{"lab1val1", "lab3val2"}},
			},
		},
		{
			description: "Case 2: value at 2 array levels and value at array output",
			metricKey:   "l1.l2b.*.l3a.l4a.*.label5",
			labelKeys: []config.Label{
				{Alias: "label1", Path: "l1.l2a.label1"},
				{Alias: "label2", Path: "l1.l2b.*.l3a.label2"},
				{Alias: "label4", Path: "l1.l2b.*.l3a.l4a.*.label4"},
			},
			expected: []ExpectedMetric{
				{Metric: "1", Labels: []string{"lab1val1", "lab2val1", "lab4val1"}},
				{Metric: "2", Labels: []string{"lab1val1", "lab2val2", "lab4val2"}},
			},
		},
		{
			description: "Case 2bis: value with 3 array levels",
			metricKey:   "l1.l2b.*.l3a.l4a.*.label5",
			labelKeys: []config.Label{
				{Alias: "label1", Path: "l1.l2a.label1"},
				{Alias: "label2", Path: "l1.l2b.*.l3a.label2"},
				{Alias: "label4", Path: "l1.l2b.*.l3a.l4a.*.label4"},
			},
			expected: []ExpectedMetric{
				{Metric: "1", Labels: []string{"lab1val1", "lab2val1", "lab4val1"}},
				{Metric: "2", Labels: []string{"lab1val1", "lab2val2", "lab4val2"}},
			},
		},
		{
			description: "Case 3: test with a non-existent label",
			metricKey:   "l1.l2b.*.l3a.label2",
			labelKeys: []config.Label{
				{Alias: "l2a_labelx", Path: "l1.l2a.labelx"},
				{Alias: "label3", Path: "l1.l2b.*.l3a.label3"},
				{Alias: "l3a_labelx", Path: "l1.l2b.*.l3a.labelx"},
			},
			expected: []ExpectedMetric{
				{Metric: "lab2val1", Labels: []string{"", "lab3val1", ""}},
				{Metric: "lab2val2", Labels: []string{"", "lab3val2", ""}},
			},
		},
		{
			description: "Case 4: test without labels",
			metricKey:   "l1.l2b.*.l3a.label2",
			labelKeys:   []config.Label{},
			expected: []ExpectedMetric{
				{Metric: "lab2val1", Labels: []string{}},
				{Metric: "lab2val2", Labels: []string{}},
			},
		},
		{
			description: "Cas 5: label ordering impact",
			metricKey:   "l1.l2b.*.l3a.label2",
			labelKeys: []config.Label{
				{Alias: "l2a_labelx", Path: "l1.l2b.*.l3a.label3"},
				{Alias: "l2a_labelx", Path: "l1.l2a.label1"},
			},
			expected: []ExpectedMetric{
				{Metric: "lab2val1", Labels: []string{"lab1val1", "lab3val1"}},
				{Metric: "lab2val2", Labels: []string{"lab1val1", "lab3val2"}},
			},
		},
	}

	// Test execution
	for _, test := range tests {
		// Map to track validated items
		validated := make(map[int]bool)

		extractor, err := NewExtractor(".", test.metricKey, test.labelKeys)
		assert.NoError(t, err)
		// Call function with decoded JSON, path, labels, and callback
		extractor.ExtractMetrics(data, func(value string, labels []string) {
			// Checking the value found against expectations
			callbackTester(value, labels, test.expected, validated, t)
		})

		// Check that each expected element has been validated at least once
		for i, exp := range test.expected {
			if !validated[i] {
				t.Errorf("Callback call not expected : Value='%s', Labels='%v'", exp.Metric, exp.Labels)
			}
		}
	}
}
