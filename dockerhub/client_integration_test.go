package dockerhub_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gopher-lab/gopher-updater/dockerhub"
)

var _ = Describe("Client Integration", func() {
	var (
		mux    *http.ServeMux
		server *httptest.Server
		client *dockerhub.Client
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mux = http.NewServeMux()
		server = httptest.NewServer(mux)

		client = dockerhub.NewClient("user", "pass", server.Client())
		client.AuthBaseURL = server.URL
		client.RegistryBaseURL = server.URL

		// Mock DockerHub Auth
		mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/token" {
				http.NotFound(w, r)
				return
			}
			user, pass, ok := r.BasicAuth()
			Expect(ok).To(BeTrue())
			Expect(user).To(Equal("user"))
			Expect(pass).To(Equal("pass"))
			fmt.Fprint(w, `{"token":"a-dummy-token"}`)
		})
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("TagExists", func() {
		It("should return true if the manifest exists (200 OK)", func() {
			mux.HandleFunc("/v2/my/repo/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodHead))
				Expect(r.Header.Get("Authorization")).To(Equal("Bearer a-dummy-token"))
				w.WriteHeader(http.StatusOK)
			})

			exists, err := client.TagExists(ctx, "my/repo", "latest")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false if the manifest does not exist (404 Not Found)", func() {
			mux.HandleFunc("/v2/my/repo/manifests/nonexistent", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})

			exists, err := client.TagExists(ctx, "my/repo", "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("RetagImage", func() {
		It("should successfully get and put the manifest to retag an image", func() {
			const manifestContent = `{"hello":"world"}`
			const manifestContentType = "application/vnd.docker.distribution.manifest.v2+json"

			// Handle GET for the source tag
			mux.HandleFunc("/v2/my/repo/manifests/source-tag", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodGet))
				w.Header().Set("Content-Type", manifestContentType)
				fmt.Fprint(w, manifestContent)
			})

			// Handle PUT for the target tag
			mux.HandleFunc("/v2/my/repo/manifests/target-tag", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodPut))
				Expect(r.Header.Get("Content-Type")).To(Equal(manifestContentType))

				body := make([]byte, len(manifestContent))
				_, _ = r.Body.Read(body)
				Expect(string(body)).To(Equal(manifestContent))

				w.WriteHeader(http.StatusCreated)
			})

			err := client.RetagImage(ctx, "my/repo", "source-tag", "target-tag")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if getting the source manifest fails", func() {
			mux.HandleFunc("/v2/my/repo/manifests/source-tag-fail", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})

			err := client.RetagImage(ctx, "my/repo", "source-tag-fail", "target-tag")
			Expect(err).To(HaveOccurred())
		})

		It("should return an error if putting the target manifest fails", func() {
			mux.HandleFunc("/v2/my/repo/manifests/source-tag-put-fail", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "{}")
			})
			mux.HandleFunc("/v2/my/repo/manifests/target-tag-fail", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})

			err := client.RetagImage(ctx, "my/repo", "source-tag-put-fail", "target-tag-fail")
			Expect(err).To(HaveOccurred())
		})
	})
})
