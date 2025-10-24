package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/gopher-lab/gopher-updater/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		os.Clearenv()
	})

	Context("when creating a new config", func() {
		It("should return an error if dry run is false and dockerhub user is not set", func() {
			Expect(os.Setenv("DOCKERHUB_PASSWORD", "password")).ToNot(HaveOccurred())
			Expect(os.Setenv("REPO_PATH", "repo")).ToNot(HaveOccurred())
			Expect(os.Setenv("TARGET_PREFIX", "prefix")).ToNot(HaveOccurred())
			_, err := config.New(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error if dry run is false and dockerhub password is not set", func() {
			Expect(os.Setenv("DOCKERHUB_USER", "user")).ToNot(HaveOccurred())
			Expect(os.Setenv("REPO_PATH", "repo")).ToNot(HaveOccurred())
			Expect(os.Setenv("TARGET_PREFIX", "prefix")).ToNot(HaveOccurred())
			_, err := config.New(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error if dry run is true and dockerhub credentials are not set", func() {
			Expect(os.Setenv("DRY_RUN", "true")).ToNot(HaveOccurred())
			Expect(os.Setenv("REPO_PATH", "repo")).ToNot(HaveOccurred())
			Expect(os.Setenv("TARGET_PREFIX", "prefix")).ToNot(HaveOccurred())
			_, err := config.New(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return an error if dry run is false and dockerhub credentials are set", func() {
			Expect(os.Setenv("DOCKERHUB_USER", "user")).ToNot(HaveOccurred())
			Expect(os.Setenv("DOCKERHUB_PASSWORD", "password")).ToNot(HaveOccurred())
			Expect(os.Setenv("REPO_PATH", "repo")).ToNot(HaveOccurred())
			Expect(os.Setenv("TARGET_PREFIX", "prefix")).ToNot(HaveOccurred())
			_, err := config.New(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
