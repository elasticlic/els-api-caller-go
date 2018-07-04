package els

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Access Key Test Suite", func() {

	var (
		id                = AccessKeyID("id")
		sac               = SecretAccessKey("sac")
		email      string = "example@test.com"
		now               = time.Now()
		past              = now.Add(-1 * time.Second)
		near              = time.Minute
		nearFuture        = now.Add(near)
		in                = time.Minute
		sut        *AccessKey
		bResult    bool
	)

	Describe("AccessKey", func() {

		BeforeEach(func() {
			now = time.Now()
			past = now.Add(-1 * time.Second)
			near = time.Minute
			nearFuture = now.Add(near)
			in = time.Minute
			sut = &AccessKey{
				ID:              id,
				SecretAccessKey: sac,
				ExpiryDate:      nearFuture,
				Email:           email,
			}
		})
		Describe("ValidUntil", func() {
			JustBeforeEach(func() {
				bResult = sut.ValidUntil(now, in)
			})
			Context("The key has already expired", func() {
				BeforeEach(func() {
					sut.ExpiryDate = past
				})
				It("returns false", func() {
					Expect(bResult).To(BeFalse())
				})
			})
			Context("The key will expire in the near future", func() {
				BeforeEach(func() {
					sut.ExpiryDate = nearFuture
				})
				It("returns false", func() {
					Expect(bResult).To(BeFalse())
				})
			})
			Context("The key will not expire in the near future", func() {
				BeforeEach(func() {
					sut.ExpiryDate = now.Add(near + time.Second)
				})
				It("returns true", func() {
					Expect(bResult).To(BeTrue())
				})
			})
			Context("The key never expires", func() {
				BeforeEach(func() {
					sut.ExpiryDate = time.Time{}
				})
				It("returns true", func() {
					Expect(bResult).To(BeTrue())
				})
			})
		})
		Describe("CanSign", func() {
			JustBeforeEach(func() {
				bResult = sut.CanSign()
			})
			Context("The Key is has an ID and Secret AccessKey", func() {
				It("returns false", func() {
					Expect(bResult).To(BeTrue())
				})
			})
			Context("The ID is not set", func() {
				BeforeEach(func() {
					sut.ID = ""
				})
				It("returns false", func() {
					Expect(bResult).To(BeFalse())
				})
			})
			Context("The SecretAccessKey is not set", func() {
				BeforeEach(func() {
					sut.SecretAccessKey = ""
				})
				It("returns false", func() {
					Expect(bResult).To(BeFalse())
				})
			})
		})
	})
})
