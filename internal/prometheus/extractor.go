package prometheus

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vinted/graphql-exporter/internal/config"
)

type Extractor struct {
	separator     string
	labelKeyParts [][]string
	labels        []config.Label
	valueKeyParts []string
	valueKey      string
}

func NewExtractor(separator, valueKey string, labelKeys []config.Label) (Extractor, error) {
	sortedLabelKeys := sortPaths(separator, labelKeys)
	labelKeyParts := make([][]string, len(labelKeys))
	for i, keyPath := range sortedLabelKeys {
		labelKeyParts[i] = strings.Split(keyPath.Path, ".")
	}
	valueKeyParts := strings.Split(valueKey, ".")
	e := Extractor{
		separator:     separator,
		valueKeyParts: valueKeyParts,
		labelKeyParts: labelKeyParts,
		valueKey:      valueKey,
		labels:        sortedLabelKeys,
	}

	err := e.validateLabelKeysPath()
	return e, err
}

// Type for callback function.
type CallbackFunc func(value string, labels []string)

func (e *Extractor) GetSortedLabels() []config.Label {
	return e.labels
}

// sortPaths sorts paths by length and alphabetically
func sortPaths(separator string, labels []config.Label) []config.Label {
	// Use sort.Slice to customize sort order
	sort.Slice(labels, func(i, j int) bool {
		partsI := strings.Split(labels[i].Path, separator)
		partsJ := strings.Split(labels[j].Path, separator)

		// Sort by length (number of segments)
		if len(partsI) != len(partsJ) {
			return len(partsI) < len(partsJ)
		}

		// Sort segments alphabetically
		for k := 0; k < len(partsI); k++ {
			if partsI[k] != partsJ[k] {
				return partsI[k] < partsJ[k]
			}
		}

		// Paths are identical
		return false
	})

	return labels
}

// JSON decoding function that transforms a []byte into an interface{}.
func extractGraph(data []byte) (interface{}, error) {
	var decoded interface{}
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, fmt.Errorf("fail to decode JSON : %v", err)
	}
	return decoded, nil
}

// Function that validates the `labelKeys` against the `valueKey`.
func (e *Extractor) validateLabelKeysPath() error {
	for idx, labelKeyParts := range e.labelKeyParts {

		// Rule 1: The first keyPart of each label must be the same as that of valueKey
		if e.valueKeyParts[0] != labelKeyParts[0] {
			return fmt.Errorf("the first segment of the labelKey '%s' does not match that of the valueKey '%s'", e.labels[idx].Alias, e.valueKey)
		}

		// Rule 2: A labelKey must have the same number or fewer * as the valueKey.
		// Count '*' in labelKeyParts
		labelStarCount := countStars(labelKeyParts)
		valueStarCount := countStars(e.valueKeyParts)

		if labelStarCount > valueStarCount {
			return fmt.Errorf("labelKey '%s' has more '*' than valueKey '%s'", e.labels[idx].Alias, e.valueKey)
		}

		// Rule 3: A `*` in a labelKey must have the same position in the valueKey p
		for i := 0; i < len(labelKeyParts); i++ {
			// If the labelKey has a * at this position, it can correspond to any segment of the valueKey
			if labelKeyParts[i] == "*" {
				// Check that this position is valid in the valueKey (do not exceed the length of the valueKey)
				if i >= len(e.valueKeyParts) {
					return fmt.Errorf("a '*' in labelKey '%s' exceeds the length of valueKey '%s", e.labels[idx], e.valueKey)
				}
				// Check that the '*' is in the same position as in the valueKey
				if e.valueKeyParts[i] != "*" && len(e.valueKeyParts) > i && e.valueKeyParts[i] != labelKeyParts[i] {
					return fmt.Errorf("the '*' in the labelKey '%s' is incorrectly positioned in relation to the valueKey '%s'", e.labels[idx].Path, e.valueKey)
				}
			} else {
				// If it's not an *, it must correspond exactly to the valueKey segment.
				if i >= len(e.valueKeyParts) || e.valueKeyParts[i] != labelKeyParts[i] {
					// Rule 4: After the first segment, the other segments can be different unless there is a *.
					if i > 0 && (len(e.valueKeyParts) <= i || labelKeyParts[i] != e.valueKeyParts[i]) {
						continue
					}
					return fmt.Errorf("the '%s' segment of labelKey '%s' does not match that of valueKey '%s' at index %d", labelKeyParts[i], e.labels[idx].Alias, e.valueKey, i)
				}
			}
		}

		// Rule 5: A * in labelKey cannot have an additional position with respect to valueKey
		if len(labelKeyParts) > len(e.valueKeyParts) {
			// If the labelKey has more segments than the valueKey, it is valid as long as it does not contain a misplaced *.
			if labelStarCount < valueStarCount {
				return fmt.Errorf("labelKey '%s' has more segments than valueKey '%s' with an incorrectly positioned '*'", e.labels[idx].Alias, e.valueKey)
			}
		}
	}
	return nil
}

// Function to count the number of '*' in an array
func countStars(parts []string) int {
	count := 0
	for _, part := range parts {
		if part == "*" {
			count++
		}
	}
	return count
}

// extractMetrics cans 'data' to extract metrics and labels using 'metricKey' and 'labelKeys
func (e *Extractor) ExtractMetrics(data interface{}, callback CallbackFunc) {
	// Metrics is added to data graph to be extracted
	keyPartsPaths := append(e.labelKeyParts, e.valueKeyParts)

	// Initial call to recursive function
	extractRecursive(data, keyPartsPaths, 0, make([]string, len(keyPartsPaths)), callback)
}

// Fonction extractRecursive
func extractRecursive(data interface{}, labelKeyParts [][]string, idx int, currentLabels []string, callback CallbackFunc) {

	for labelKeyIdx, labelKeyPart := range labelKeyParts {
		if len(labelKeyPart) <= idx || currentLabels[labelKeyIdx] != "" {
			continue
		}

		currentLabelKey := labelKeyPart[idx]
		if idx == len(labelKeyPart)-1 || (labelKeyIdx < len(labelKeyParts)-1 && currentLabelKey != labelKeyParts[labelKeyIdx+1][idx]) {
			extractRemainingLabelValue(data, labelKeyPart, idx, labelKeyIdx, currentLabels)
			if labelKeyIdx == len(labelKeyParts)-1 {
				callback(fmt.Sprintf("%v", currentLabels[len(currentLabels)-1]), currentLabels[:len(currentLabels)-1])
			}
			continue
		}
		if labelKeyIdx == len(labelKeyParts)-1 {
			recurse(data, currentLabelKey, labelKeyParts, idx+1, currentLabels, callback)

		}
	}

}

func recurse(data interface{}, currentLabelKey string, labelKeyParts [][]string, idx int, currentLabels []string, callback CallbackFunc) {
	if currentLabelKey == "*" {
		if dataArray, ok := data.([]interface{}); ok {
			for _, item := range dataArray {
				var newLabels []string
				newLabels = append(newLabels, currentLabels...)
				extractRecursive(item, labelKeyParts, idx, newLabels, callback)
			}
		}
	} else {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if nextData, exists := dataMap[currentLabelKey]; exists {
				extractRecursive(nextData, labelKeyParts, idx, currentLabels, callback)
			}
		}
	}
}

// extractRemainingLabelValue extracts the value of a labelKey
func extractRemainingLabelValue(data interface{}, labelKeyPart []string, idx, labelKeyIdx int, currentLabels []string) {
	var value interface{} = data
	for i := idx; i < len(labelKeyPart); i++ {
		switch val := value.(type) {
		case map[string]interface{}:
			if nextValue, exists := val[labelKeyPart[i]]; exists {
				value = nextValue
			} else {
				return
			}
		case []interface{}:
			return
		default:
			value = val
		}

	}

	currentLabels[labelKeyIdx] = fmt.Sprintf("%v", value)
}
