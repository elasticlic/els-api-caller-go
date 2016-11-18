package els

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sign Test Suite", func() {

	var (
		keyID           string          = "AccessKeyID"
		sac             string          = "secretAccessKey"
		AccessKeyID     AccessKeyID     = AccessKeyID(keyID)
		secretAccessKey SecretAccessKey = SecretAccessKey(sac)
		email           string          = "example@test.com"
		sut             *APISigner
		err             error
		now, _          = time.Parse(time.RFC3339, "2015-01-01T00:00:00Z")
		utcStr          = now.UTC().Format(time.RFC3339)
		future          = now.Add(1 * time.Hour)
		t               time.Time
		k               *AccessKey
		r               *http.Request
		method          = "POST"
		vPrefix         string
		route           = "/path/to/route"
		query           = "?query1&query2"
		json            = []byte(`{"title":"ATitle"}`)
		bodyBuffer      = bytes.NewBuffer(json)
		body            io.Reader

		buildRequest = func() {
			r, err = http.NewRequest(method, vPrefix+route+query, body)
			Expect(err).To(BeNil())
		}

		expectedAuth = func() string {
			ss := method + "\n"

			if body != nil {
				ss += fmt.Sprintf("%x\n", md5.Sum(json))
				ss += "application/json;charset=utf-8\n"
			} else {
				ss += "\n\n"
			}

			ss += utcStr + "\n"
			ss += vPrefix + route
			h := hmac.New(sha256.New, []byte(sac))
			h.Write([]byte(ss))
			hStr := base64.StdEncoding.EncodeToString(h.Sum(nil))
			a := "ELS " + keyID + ":" + hStr
			return a
		}
	)

	BeforeEach(func() {
		t = now
		vPrefix = "/1.0"
		k = &AccessKey{
			ID:              AccessKeyID,
			SecretAccessKey: secretAccessKey,
			ExpiryDate:      future,
			Email:           email,
		}
	})

	Describe("NewAPISigner", func() {

		JustBeforeEach(func() {
			sut, err = NewAPISigner(k)
		})
		It("Creates a new APISigner", func() {
			Expect(err).To(BeNil())
			Expect(sut).NotTo(BeNil())
		})
		Context("A nil pointer is passed", func() {
			BeforeEach(func() {
				k = nil
			})
			It("Returns ErrNoAccessKey", func() {
				Expect(err).To(Equal(ErrNoAccessKey))
			})
		})
	})

	Describe("APISigner", func() {
		BeforeEach(func() {
			body = bodyBuffer
			sut, err = NewAPISigner(k)
			Expect(err).To(BeNil())
			buildRequest()
		})

		Describe("Sign", func() {
			JustBeforeEach(func() {
				err = sut.Sign(r, now)
			})

			Context("The request has a body", func() {
				It("signs the request correctly and leaves the body intact", func() {
					Expect(err).To(BeNil())
					a := r.Header.Get("Authorization")
					Expect(a).To(Equal(expectedAuth()))
					a = r.Header.Get("X-Els-Date")
					Expect(a).To(Equal(utcStr))
					b, err := ioutil.ReadAll(r.Body)
					Expect(err).To(BeNil())
					Expect(b).To(Equal(json))
				})
			})

			Context("The request has no body", func() {
				BeforeEach(func() {
					body = nil
					buildRequest()
				})
				It("signs the request correctly", func() {
					Expect(err).To(BeNil())
					a := r.Header.Get("Authorization")
					Expect(a).To(Equal(expectedAuth()))
					a = r.Header.Get("X-Els-Date")
					Expect(a).To(Equal(utcStr))
				})
			})

			Context("No request is passed", func() {
				BeforeEach(func() {
					r = nil
				})
				It("returns ErrNoRequest", func() {
					Expect(err).To(Equal(ErrNoRequest))
				})
			})

			Context("The path to sign does not begin with a valid version of the API", func() {
				BeforeEach(func() {
					vPrefix = "/invalid"
					buildRequest()
				})
				It("returns ErrNoRequest", func() {
					Expect(err).To(Equal(ErrRequestInvalidURL))
				})
			})

			Context("The access key has expired", func() {
				BeforeEach(func() {
					sut.accessKey.ExpiryDate = now
				})
				It("returns ErrExpiredAccessKey", func() {
					Expect(err).To(Equal(ErrExpiredAccessKey))
				})
			})
		})
	})
})
