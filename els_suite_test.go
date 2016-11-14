package els

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMatcher(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "els Suite")
}
