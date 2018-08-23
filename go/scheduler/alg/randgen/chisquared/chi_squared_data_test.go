package chisquared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCriticalValue(t *testing.T) {
	crit, err := CriticalValue(52, 0.999)
	assert.NoError(t, err, "CriticalValue lookup failed")
	assert.InEpsilon(t, 89.272, crit, 1.0e-12, "Chi-squared critical value at DoF=52, prob=0.999 wrong")

	_, err = CriticalValue(10000, 0.999)
	assert.NotNil(t, err, "Should have failed with too-large DoF")

	for _, prob := range []float64{1.1, 0.5, 3, -10} {
		_, err = CriticalValue(100, prob)
		assert.NotNilf(t, err, "Should have failed with crazy probability %g", prob)
	}
}