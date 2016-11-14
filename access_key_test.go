package els

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Access Key Test Suite", func() {

	var (
		id         = AccessKeyId("id")
		sac        = SecretAccessKey("sac")
		isValid    bool
		email      string = "example@test.com"
		now               = time.Now()
		past              = now.Add(-1 * time.Second)
		near              = time.Minute
		nearFuture        = now.Add(near)
		in                = time.Minute
		sut        *AccessKey
	)

	Describe("AccessKey", func() {
		BeforeEach(func() {
			sut = &AccessKey{
				Id:              id,
				SecretAccessKey: sac,
				ExpiryDate:      nearFuture,
				Email:           email,
			}
		})
		Describe("ExpiresIn", func() {
			JustBeforeEach(func() {
				isValid = sut.ValidUntil(now, in)
			})
			Context("The key has already expired", func() {
				BeforeEach(func() {
					sut.ExpiryDate = past
				})
				It("returns false", func() {
					Expect(isValid).To(BeFalse())
				})
			})
		})
	})
})
