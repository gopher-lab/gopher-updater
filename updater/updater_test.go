package updater_test

import (
	"context"
	"errors"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gopher-lab/gopher-updater/config"
	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/updater"
)

var _ = Describe("Updater", func() {
	var (
		up                  *updater.Updater
		mockCosmosClient    *MockCosmosClient
		mockDockerHubClient *MockDockerHubClient
		cfg                 *config.Config
		ctx                 context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCosmosClient = &MockCosmosClient{}
		mockDockerHubClient = &MockDockerHubClient{}
		cfg = &config.Config{
			RepoPath:     "my/repo",
			SourcePrefix: "release-",
			TargetPrefix: "mainnet-",
		}

		up = updater.New(mockCosmosClient, mockDockerHubClient, cfg)
	})

	Context("when processing upgrades", func() {
		It("should retag the image if a single upgrade height has been reached and the tag does not exist", func() {
			plans := []cosmos.Plan{{Name: "v1.2.3", Height: "100"}}
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return plans, nil
			}
			mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
				return 100, nil
			}
			mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
				Expect(tag).To(Equal("mainnet-v1.2.3"))
				return false, nil
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())

			retagCalls := mockDockerHubClient.RetagCalls()
			Expect(retagCalls).To(HaveLen(1))
			Expect(retagCalls[0].SourceTag).To(Equal("release-v1.2.3"))
			Expect(retagCalls[0].TargetTag).To(Equal("mainnet-v1.2.3"))
		})

		It("should process the oldest of multiple pending upgrades", func() {
			plans := []cosmos.Plan{
				{Name: "v1.2.4", Height: "110"},
				{Name: "v1.2.3", Height: "100"},
			}
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return plans, nil
			}
			mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
				return 111, nil
			}
			mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
				return false, nil // Neither tag exists
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())

			retagCalls := mockDockerHubClient.RetagCalls()
			Expect(retagCalls).To(HaveLen(1))
			Expect(retagCalls[0].SourceTag).To(Equal("release-v1.2.3")) // Processes the one with lower height
			Expect(retagCalls[0].TargetTag).To(Equal("mainnet-v1.2.3"))
		})

		It("should process the next pending upgrade if one was already applied", func() {
			plans := []cosmos.Plan{
				{Name: "v1.2.4", Height: "110"},
				{Name: "v1.2.3", Height: "100"},
			}
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return plans, nil
			}
			mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
				return 111, nil
			}
			mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
				if tag == "mainnet-v1.2.3" {
					return true, nil // This one is already done
				}
				return false, nil
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())

			retagCalls := mockDockerHubClient.RetagCalls()
			Expect(retagCalls).To(HaveLen(1))
			Expect(retagCalls[0].SourceTag).To(Equal("release-v1.2.4")) // Processes the next one
			Expect(retagCalls[0].TargetTag).To(Equal("mainnet-v1.2.4"))
		})

		It("should do nothing if the upgrade height has not been reached", func() {
			plans := []cosmos.Plan{{Name: "v1.2.3", Height: "100"}}
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return plans, nil
			}
			mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
				return 98, nil
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockDockerHubClient.RetagCalls()).To(BeEmpty())
		})

		It("should do nothing if the target tag already exists", func() {
			plans := []cosmos.Plan{{Name: "v1.2.3", Height: "100"}}
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return plans, nil
			}
			mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
				return 101, nil
			}
			mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
				return true, nil
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockDockerHubClient.RetagCalls()).To(BeEmpty())
		})

		It("should do nothing if there are no passed upgrade proposals", func() {
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return []cosmos.Plan{}, nil
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockDockerHubClient.RetagCalls()).To(BeEmpty())
		})

		It("should return an error if getting upgrade plans fails", func() {
			mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
				return nil, errors.New("cosmos boom")
			}

			err := up.CheckAndProcessUpgrade(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cosmos boom"))
		})

		Context("with dry run enabled", func() {
			BeforeEach(func() {
				cfg.DryRun = true
			})

			It("should not retag the image if a single upgrade height has been reached and the tag does not exist", func() {
				plans := []cosmos.Plan{{Name: "v1.2.3", Height: "100"}}
				mockCosmosClient.getUpgradePlansFunc = func(ctx context.Context) ([]cosmos.Plan, error) {
					return plans, nil
				}
				mockCosmosClient.getLatestBlockHeightFunc = func(ctx context.Context) (int64, error) {
					return 100, nil
				}
				mockDockerHubClient.tagExistsFunc = func(ctx context.Context, repoPath, tag string) (bool, error) {
					Expect(tag).To(Equal("mainnet-v1.2.3"))
					return false, nil
				}

				err := up.CheckAndProcessUpgrade(ctx)
				Expect(err).ToNot(HaveOccurred())

				retagCalls := mockDockerHubClient.RetagCalls()
				Expect(retagCalls).To(HaveLen(0))
			})
		})
	})
})

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
	retagCalls    []RetagCall
	tagExistsFunc func(ctx context.Context, repoPath, tag string) (bool, error)
}

type RetagCall struct {
	RepoPath  string
	SourceTag string
	TargetTag string
}

func (m *MockDockerHubClient) RetagImage(ctx context.Context, repoPath, sourceTag, targetTag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retagCalls = append(m.retagCalls, RetagCall{RepoPath: repoPath, SourceTag: sourceTag, TargetTag: targetTag})
	return nil
}

func (m *MockDockerHubClient) TagExists(ctx context.Context, repoPath, tag string) (bool, error) {
	if m.tagExistsFunc != nil {
		return m.tagExistsFunc(ctx, repoPath, tag)
	}
	return false, nil
}

func (m *MockDockerHubClient) RetagCalls() []RetagCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.retagCalls
}
