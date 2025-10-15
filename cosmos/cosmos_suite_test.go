package cosmos_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCosmos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cosmos Suite")
}
