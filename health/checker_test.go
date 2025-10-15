package health_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/health"
)

func TestHealth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Health Suite")
}

var _ = Describe("Checker", func() {
	var (
		checker             *health.Checker
		mockCosmosClient    *MockCosmosClient
		mockDockerHubClient *MockDockerHubClient
		ctx                 context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCosmosClient = &MockCosmosClient{}
		mockDockerHubClient = &MockDockerHubClient{}
		checker = health.NewChecker(mockCosmosClient, mockDockerHubClient, "my/repo")
	})

	It("should return no error when both clients are healthy", func() {
		mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
			return 1, nil
		}
		mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
			return false, nil
		}

		err := checker.Ready(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return an error if the cosmos client fails", func() {
		mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
			return 0, errors.New("cosmos boom")
		}
		mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
			return false, nil
		}

		err := checker.Ready(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cosmos connection failed"))
	})

	It("should return an error if the dockerhub client fails", func() {
		mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
			return 1, nil
		}
		mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
			return false, errors.New("dockerhub boom")
		}

		err := checker.Ready(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dockerhub connection failed"))
	})
})

// --- Mock Implementations ---
// NOTE: These are duplicated from updater/updater_test.go because Go does not
// allow sharing test code between packages.

// MockCosmosClient is a mock implementation of the Cosmos client for testing.
type MockCosmosClient struct {
	getUpgradePlansFunc      func(ctx context.Context) ([]cosmos.Plan, error)
	getLatestBlockHeightFunc func(ctx context.Context) (int64, error)
}

func (m *MockCosmosClient) GetUpgradePlans(ctx context.Context) ([]cosmos.Plan, error) {
	if m.getUpgradePlansFunc != nil {
		return m.getUpgradePlansFunc(ctx)
	}
	return nil, nil
}

func (m *MockCosmosClient) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	if m.getLatestBlockHeightFunc != nil {
		return m.getLatestBlockHeightFunc(ctx)
	}
	return 0, nil
}

// MockDockerHubClient is a mock implementation of the DockerHub client for testing.
type MockDockerHubClient struct {
	mu            sync.Mutex
	retagCalls    []any
	tagExistsFunc func(ctx context.Context, repoPath, tag string) (bool, error)
}

func (m *MockDockerHubClient) RetagImage(ctx context.Context, repoPath, sourceTag, targetTag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retagCalls = append(m.retagCalls, nil) // Simplified for this test
	return nil
}

func (m *MockDockerHubClient) TagExists(ctx context.Context, repoPath, tag string) (bool, error) {
	if m.tagExistsFunc != nil {
		return m.tagExistsFunc(ctx, repoPath, tag)
	}
	return false, nil
}
