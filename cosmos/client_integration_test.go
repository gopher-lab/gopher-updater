package cosmos_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gopher-lab/gopher-updater/cosmos"
)

var _ = Describe("Client Integration", func() {
	var (
		mux    *http.ServeMux
		server *httptest.Server
		client *cosmos.Client
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mux = http.NewServeMux()
		server = httptest.NewServer(mux)
		client = cosmos.NewClient(server.URL, server.Client())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("GetLatestBlockHeight", func() {
		It("should return the correct block height on a valid response", func() {
			mux.HandleFunc("/blocks/latest", func(w http.ResponseWriter, r *http.Request) {
				_, err := fmt.Fprint(w, `{"block":{"header":{"height":"12345"}}}`)
				Expect(err).NotTo(HaveOccurred())
			})

			height, err := client.GetLatestBlockHeight(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(height).To(BeEquivalentTo(12345))
		})

		It("should return an error on a non-200 status code", func() {
			mux.HandleFunc("/blocks/latest", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})

			_, err := client.GetLatestBlockHeight(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error on malformed JSON", func() {
			mux.HandleFunc("/blocks/latest", func(w http.ResponseWriter, r *http.Request) {
				_, err := fmt.Fprint(w, `{"block":{"header":{"height":malformed}}}`)
				Expect(err).NotTo(HaveOccurred())
			})

			_, err := client.GetLatestBlockHeight(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetUpgradePlans", func() {
		It("should correctly parse and filter for passed software upgrade proposals", func() {
			mux.HandleFunc("/cosmos/gov/v1beta1/proposals", func(w http.ResponseWriter, r *http.Request) {
				_, err := fmt.Fprint(w, `{
					"proposals": [
						{
							"status": "PROPOSAL_STATUS_PASSED",
							"content": {
								"@type": "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal",
								"plan": { "name": "v1.2.3", "height": "100" }
							}
						},
						{
							"status": "PROPOSAL_STATUS_REJECTED",
							"content": {
								"@type": "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal",
								"plan": { "name": "v1.2.4", "height": "200" }
							}
						},
						{
							"status": "PROPOSAL_STATUS_PASSED",
							"content": {
								"@type": "/cosmos.params.v1beta1.ParameterChangeProposal"
							}
						}
					]
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			plans, err := client.GetUpgradePlans(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plans).To(HaveLen(1))
			Expect(plans[0].Name).To(Equal("v1.2.3"))
			Expect(plans[0].Height).To(Equal("100"))
		})

		It("should return an empty slice when no passed upgrade proposals are found", func() {
			mux.HandleFunc("/cosmos/gov/v1beta1/proposals", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"proposals": []}`)
			})

			plans, err := client.GetUpgradePlans(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plans).To(BeEmpty())
		})

		It("should return an error on a non-200 status code", func() {
			mux.HandleFunc("/cosmos/gov/v1beta1/proposals", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})

			_, err := client.GetUpgradePlans(ctx)
			Expect(err).To(HaveOccurred())
		})
	})
})
