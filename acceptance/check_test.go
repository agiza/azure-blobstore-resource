package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check", func() {
	var (
		container string
	)

	BeforeEach(func() {
		rand.Seed(time.Now().UTC().UnixNano())
		container = fmt.Sprintf("azureblobstore%d", rand.Int())
		createContainer(container)
	})

	AfterEach(func() {
		deleteContainer(container)
	})

	Context("when given a version", func() {
		var (
			snapshotTimestampCurrent *time.Time
			snapshotTimestampNew     *time.Time
			snapshotTimestampNewer   *time.Time
		)

		BeforeEach(func() {
			createBlobWithSnapshot(container, "example.json")
			snapshotTimestampCurrent = createBlobWithSnapshot(container, "example.json")
			snapshotTimestampNew = createBlobWithSnapshot(container, "example.json")
			snapshotTimestampNewer = createBlobWithSnapshot(container, "example.json")
		})

		It("returns all versions since blob snapshot version", func() {
			check := exec.Command(pathToCheck)
			check.Stderr = os.Stderr

			stdin, err := check.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"versioned_file": "example.json"
					},
					"version": { "snapshot": %q }
				}`,
				config.StorageAccountName,
				config.StorageAccountKey,
				container,
				snapshotTimestampCurrent.Format(time.RFC3339Nano),
			))
			Expect(err).NotTo(HaveOccurred())

			output, err := check.Output()
			Expect(err).NotTo(HaveOccurred())

			var versions []struct {
				Path     *string    `json:"path"`
				Version  *string    `json:"version"`
				Snapshot *time.Time `json:"snapshot"`
			}
			err = json.Unmarshal(output, &versions)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(versions)).To(Equal(3))
			Expect(versions[0].Path).To(BeNil())
			Expect(versions[0].Version).To(BeNil())
			Expect(versions[0].Snapshot).To(Equal(snapshotTimestampCurrent))
			Expect(versions[1].Path).To(BeNil())
			Expect(versions[1].Version).To(BeNil())
			Expect(versions[1].Snapshot).To(Equal(snapshotTimestampNew))
			Expect(versions[2].Path).To(BeNil())
			Expect(versions[2].Version).To(BeNil())
			Expect(versions[2].Snapshot).To(Equal(snapshotTimestampNewer))
		})
	})

	Context("when blob doesn't have a snapshot", func() {
		BeforeEach(func() {
			createBlob(container, "example.json")
		})

		It("returns a zero timestamp version", func() {
			check := exec.Command(pathToCheck)
			check.Stderr = os.Stderr

			stdin, err := check.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"versioned_file": "example.json"
					}
				}`,
				config.StorageAccountName,
				config.StorageAccountKey,
				container,
			))
			Expect(err).NotTo(HaveOccurred())

			output, err := check.Output()
			Expect(err).NotTo(HaveOccurred())

			var versions []struct {
				Path     *string    `json:"path"`
				Version  *string    `json:"version"`
				Snapshot *time.Time `json:"snapshot"`
			}
			err = json.Unmarshal(output, &versions)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(versions)).To(Equal(1))
			Expect(versions[0].Snapshot).To(Equal(&time.Time{}))
			Expect(versions[0].Path).To(BeNil())
			Expect(versions[0].Version).To(BeNil())
		})
	})

	Context("when there is no blob", func() {
		It("returns an error", func() {
			check := exec.Command(pathToCheck)
			check.Stderr = os.Stderr

			stdin, err := check.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"versioned_file": "example.json"
					},
					"version": { "snapshot": "2017-08-08T23:27:16.2942812Z" }
				}`,
				config.StorageAccountName,
				config.StorageAccountKey,
				container,
			))
			Expect(err).NotTo(HaveOccurred())

			var stderr bytes.Buffer
			check.Stderr = &stderr

			err = check.Run()
			Expect(err).To(HaveOccurred())

			Expect(stderr.String()).To(ContainSubstring("failed to find blob: example.json"))
		})

		Context("when the initial_version is provided with", func() {
			var (
				initialVersionTime *time.Time
			)

			BeforeEach(func() {
				initialVersionTime = timePtr(time.Date(2017, time.January, 02, 01, 01, 01, 01, time.UTC))
			})

			It("returns the initial_version as the version", func() {
				check := exec.Command(pathToCheck)
				check.Stderr = os.Stderr

				stdin, err := check.StdinPipe()
				Expect(err).NotTo(HaveOccurred())

				_, err = io.WriteString(stdin, fmt.Sprintf(`{
						"source": {
							"storage_account_name": %q,
							"storage_account_key": %q,
							"container": %q,
							"versioned_file": "example.json"
						},
						"params": {
							"initial_version": %q
						},
						"version": { "snapshot": "2017-08-08T23:27:16.2942812Z" }
					}`,
					config.StorageAccountName,
					config.StorageAccountKey,
					container,
					initialVersionTime.Format(time.RFC3339Nano),
				))
				Expect(err).NotTo(HaveOccurred())

				output, err := check.Output()
				Expect(err).NotTo(HaveOccurred())

				var versions []struct {
					Path     *string    `json:"path"`
					Version  *string    `json:"version"`
					Snapshot *time.Time `json:"snapshot"`
				}
				err = json.Unmarshal(output, &versions)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(versions)).To(Equal(1))
				Expect(versions[0].Snapshot).To(Equal(initialVersionTime))
				Expect(versions[0].Path).To(BeNil())
				Expect(versions[0].Version).To(BeNil())
			})
		})
	})

	Context("when a regex pattern is provided", func() {
		BeforeEach(func() {
			createBlob(container, "example-1.2.3.json")
			createBlob(container, "example-0.1.0.json")
			createBlob(container, "example-1.2.4.json")
			createBlob(container, "example-2.0.0.json")
		})

		It("returns all versions since version that matches the regexp", func() {
			check := exec.Command(pathToCheck)
			check.Stderr = os.Stderr

			stdin, err := check.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"regexp": "example-(.*).json"
					},
					"version": { "version": "1.0.0" }
				}`,
				config.StorageAccountName,
				config.StorageAccountKey,
				container,
			))
			Expect(err).NotTo(HaveOccurred())

			output, err := check.Output()
			Expect(err).NotTo(HaveOccurred())

			var versions []struct {
				Path     *string    `json:"path"`
				Version  *string    `json:"version"`
				Snapshot *time.Time `json:"snapshot"`
			}
			err = json.Unmarshal(output, &versions)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(versions)).To(Equal(3))
			Expect(versions[0].Path).To(Equal(stringPtr("example-1.2.3.json")))
			Expect(versions[0].Version).To(Equal(stringPtr("1.2.3")))
			Expect(versions[0].Snapshot).To(BeNil())
			Expect(versions[1].Path).To(Equal(stringPtr("example-1.2.4.json")))
			Expect(versions[1].Version).To(Equal(stringPtr("1.2.4")))
			Expect(versions[1].Snapshot).To(BeNil())
			Expect(versions[2].Path).To(Equal(stringPtr("example-2.0.0.json")))
			Expect(versions[2].Version).To(Equal(stringPtr("2.0.0")))
			Expect(versions[2].Snapshot).To(BeNil())
		})
	})

	Context("when a blob is being copied", func() {
		BeforeEach(func() {
			createBlob(container, "example-1.2.3.json")
		})

		It("returns only versions that matches the regexp which has been copied", func() {
			copyBlob(container, "example-2.3.4.json", "http://example.com")

			Eventually(func() *string {
				check := exec.Command(pathToCheck)
				check.Stderr = os.Stderr

				stdin, err := check.StdinPipe()
				Expect(err).NotTo(HaveOccurred())

				_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"regexp": "example-(.*).json"
					},
					"version": { "version": "1.0.0" }
				}`,
					config.StorageAccountName,
					config.StorageAccountKey,
					container,
				))
				Expect(err).NotTo(HaveOccurred())

				output, err := check.Output()
				Expect(err).NotTo(HaveOccurred())

				var versions []struct {
					Path     *string    `json:"path"`
					Snapshot *time.Time `json:"snapshot"`
				}
				err = json.Unmarshal(output, &versions)
				Expect(err).NotTo(HaveOccurred())
				return versions[len(versions)-1].Path
			}, 10*time.Second, time.Second).Should(Equal(stringPtr("example-2.3.4.json")))
		})

		It("returns just the latest version that matches the regexp which has been copied", func() {
			copyBlob(container, "example-2.3.4.json", "http://does.not.exist")

			Consistently(func() *string {
				check := exec.Command(pathToCheck)
				check.Stderr = os.Stderr

				stdin, err := check.StdinPipe()
				Expect(err).NotTo(HaveOccurred())

				_, err = io.WriteString(stdin, fmt.Sprintf(`{
					"source": {
						"storage_account_name": %q,
						"storage_account_key": %q,
						"container": %q,
						"regexp": "example-(.*).json"
					},
					"version": { "version": "1.0.0" }
				}`,
					config.StorageAccountName,
					config.StorageAccountKey,
					container,
				))
				Expect(err).NotTo(HaveOccurred())

				output, err := check.Output()
				Expect(err).NotTo(HaveOccurred())

				var versions []struct {
					Path     *string    `json:"path"`
					Snapshot *time.Time `json:"snapshot"`
				}
				err = json.Unmarshal(output, &versions)
				Expect(err).NotTo(HaveOccurred())
				return versions[0].Path
			}).Should(Equal(stringPtr("example-1.2.3.json")))
		})
	})
})

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
