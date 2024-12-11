package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinted/graphql-exporter/internal/config"
)

func TestGitlabGraph(t *testing.T) {
	// Jeu de données JSON
	data := `
{
    "projects": {
      "nodes": [
        {
          "name": "devops",
          "group": {
            "name": "ubbleai"
          },
          "pipelines": {
            "nodes": [
              {
                "duration": null,
                "failureReason": null,
                "jobs": {
                  "nodes": [
                    {
                      "duration": 103,
                      "name": "kics-iac-sast-2",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 35,
                      "name": "semgrep-sast-1",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 79,
                      "name": "gitlab-advanced-sast-1",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 44,
                      "name": "secret-detection-0",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 78,
                      "name": "gemnasium-python-dependency_scanning",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 114,
                      "name": "gemnasium-dependency_scanning",
                      "stage": {
                        "name": "test"
                      }
                    },
                    {
                      "duration": 26,
                      "name": "precommit",
                      "stage": {
                        "name": "pre-build"
                      }
                    }
                  ]
                }
              }
            ]
          }
        }
      ]
    }
  }		
`

	dataObject, err := extractGraph([]byte(data))
	assert.NoError(t, err)

	// Test de différents chemins et valeurs attendues
	tests := []struct {
		description string
		metricKey   string
		labelKeys   []config.Label
		expected    []ExpectedMetric
	}{
		{
			description: "Cas 1: value à 1 seul niveau de tableau",
			metricKey:   "projects.nodes.*.pipelines.nodes.*.jobs.nodes.*.duration",
			labelKeys: []config.Label{
				{Alias: "job_name", Path: "projects.nodes.*.pipelines.nodes.*.jobs.nodes.*.name"},
				{Alias: "stage_name", Path: "projects.nodes.*.pipelines.nodes.*.jobs.nodes.*.stage.name"},
				{Alias: "project_name", Path: "projects.nodes.*.name"},
				{Alias: "group_name", Path: "projects.nodes.*.group.name"},
			},
			expected: []ExpectedMetric{
				{Metric: "103", Labels: []string{"devops", "ubbleai", "kics-iac-sast-2", "test"}},
				{Metric: "35", Labels: []string{"devops", "ubbleai", "semgrep-sast-1", "test"}},
				{Metric: "79", Labels: []string{"devops", "ubbleai", "gitlab-advanced-sast-1", "test"}},
				{Metric: "44", Labels: []string{"devops", "ubbleai", "secret-detection-0", "test"}},
				{Metric: "78", Labels: []string{"devops", "ubbleai", "gemnasium-python-dependency_scanning", "test"}},
				{Metric: "114", Labels: []string{"devops", "ubbleai", "gemnasium-dependency_scanning", "test"}},
				{Metric: "26", Labels: []string{"devops", "ubbleai", "precommit", "pre-build"}},
			},
		},
	}

	// Exécution des tests
	for _, test := range tests {
		// Map pour suivre les éléments validés
		validated := make(map[int]bool)

		extractor, err := NewExtractor(".", test.metricKey, test.labelKeys)
		assert.NoError(t, err)
		// Appeler la fonction avec le JSON décodé, le chemin, les labels, et le callback
		extractor.ExtractMetrics(dataObject, func(value string, labels []string) {
			// Vérification de la valeur trouvée avec les attentes
			callbackTester(value, labels, test.expected, validated, t)
		})

		// Vérifier que chaque élément attendu a été validé au moins une fois
		for i, exp := range test.expected {
			if !validated[i] {
				t.Errorf("L'attente n'a pas été validée : Value='%s', Labels='%v'", exp.Metric, exp.Labels)
			}
		}
	}
}
