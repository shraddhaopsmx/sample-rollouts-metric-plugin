package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogs(t *testing.T) {
	t.Run("basic flow with parameters defined globally -", func(t *testing.T) {
		logsData := `
        monitoringProvider: ELASTICSEARCH
        accountName: ds-elastic
        scoringAlgorithm: Canary
        index: kubernetes*
        responseKeywords: log,message
        disableDefaultErrorTopics: false
        tags:
        - errorString: NOnOutOfMemoryError
          tag: tag1
        errorTopics:
        - errorString: OnOutOfMemoryError
          topic: CRITICAL
          type: NotDefault`

		_, err := processYamlLogs([]byte(logsData), "templateMetrics", "${namespace_key},${service},${ingress}")
		assert.Nil(t, err)
	})
}
