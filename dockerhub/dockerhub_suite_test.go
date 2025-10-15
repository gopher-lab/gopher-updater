package dockerhub_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDockerHub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DockerHub Suite")
}
