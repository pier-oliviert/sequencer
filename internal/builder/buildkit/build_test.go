package buildkit

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pier-oliviert/sequencer/internal/builder/secrets"
)

var _ = Describe("Buildkit Build", func() {
	Context("NewBuilder", func() {
		It("sets the name for the builder to use", func() {
			_, err := NewBuilder(WithCacheTags("latest"))

			Expect(err).To(BeNil())
		})
	})

	Context("WithSecrets", func() {
		It("stores the content of each key in a separate file", func() {
			keys := []secrets.KeyValue{
				{
					Key:   "test",
					Value: "Content",
				},
			}

			builder, err := NewBuilder(
				WithCacheTags("latest"),
				WithSecrets(keys),
			)

			file := builder.files[keys[0]]

			Expect(err).To(BeNil())
			Expect(file).ToNot(BeNil())
		})
	})
})
