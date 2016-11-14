package els

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("APIHandler Test Suite", func() {

	var (
		email      string = "example@test.com"
		password   string = "password"
		preHashed  bool
		err        error
		sut        *APIHandler
		statusCode int
		expDays    uint            = 3
		ctx        context.Context = context.Background()
		server     *httptest.Server
		k          *AccessKey
		reqRec     *http.Request

		simServer = func(statusCode int, body string) (*httptest.Server, *APIHandler) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqRec = r
				w.WriteHeader(statusCode)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, body)
			}))

			t := &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					return url.Parse(server.URL)
				},
			}

			c := &http.Client{Transport: t}
			h := NewAPIHandler(c)
			u, err := url.Parse(server.URL)
			if err != nil {
				log.Panic(err)
			}

			h.Scheme = u.Scheme
			h.Domain = u.Host
			return server, h
		}
		basicHash = func(email string, pw string, ph bool) string {

			if !ph {
				// ELS requires clients to pre-hash all plaintext passwords.
				// Note that this hash is *NOT* what is stored in the ELS database.
				sh := sha256.Sum256([]byte(pw))
				pw = hex.EncodeToString(sh[:])
			}

			raw := email + ":" + pw
			sig := base64.StdEncoding.EncodeToString([]byte(raw))
			h := "Basic " + sig
			return h
		}
	)

	Describe("NewAPIHandler", func() {
		It("Initialises the correct defaults", func() {
			h := NewAPIHandler(http.DefaultClient)

			Expect(h.Scheme).To(Equal("https"))
			Expect(h.Domain).To(Equal("api.elasticlicensing.com"))
			Expect(h.Version).To(Equal("1.0"))
			Expect(h.Client).To(Equal(http.DefaultClient))
		})
	})
	Describe("APIHandler", func() {
		BeforeEach(func() {
			preHashed = false
		})
		AfterEach(func() {
			if server != nil {
				server.Close()
			}
		})
		Describe("CreateAccessKey", func() {
			JustBeforeEach(func() {
				server, sut = simServer(201,
					`{
                    "accessKeyId": "anAccessKey",
                    "secretAccessKey": "aSecretAccessKey",
                    "expiryDt": "2100-01-01T00:00:00Z",
                    "emailAddress": "user@example.com"
                }`)

				k, statusCode, err = sut.CreateAccessKey(ctx, email, password, preHashed, expDays)
			})

			Context("The password has not been prehashed", func() {
				It("correctly signs the request", func() {
					u := reqRec.URL
					Expect(u.Path).To(Equal("/1.0/users/" + email + "/accessKeys"))
					Expect(u.RawQuery).To(Equal("expires=1&numDaysTillExpiry=3"))
					a := reqRec.Header["Authorization"][0]
					expected := basicHash(email, password, preHashed)
					Expect(a).Should(Equal(expected))

				})
			})

			Context("The password has been prehashed", func() {
				BeforeEach(func() {
					preHashed = true
					sh := sha256.Sum256([]byte(password))
					password = hex.EncodeToString(sh[:])
				})
				It("correctly signs the request", func() {
					u := reqRec.URL
					Expect(u.Path).To(Equal("/1.0/users/" + email + "/accessKeys"))
					Expect(u.RawQuery).To(Equal("expires=1&numDaysTillExpiry=3"))
					a := reqRec.Header["Authorization"][0]
					expected := basicHash(email, password, preHashed)
					Expect(a).Should(Equal(expected))
				})
			})

			Context("The ELS Returns an Access Key", func() {
				JustBeforeEach(func() {
					server, sut = simServer(201,
						`{
                        "accessKeyId": "anAccessKey",
                        "secretAccessKey": "aSecretAccessKey",
                        "expiryDt": "2100-01-01T00:00:00Z",
                        "emailAddress": "user@example.com"
                    }`)

					k, statusCode, err = sut.CreateAccessKey(ctx, email, password, false, expDays)
				})
				It("creates an access key", func() {
					Expect(err).To(BeNil())
					Expect(statusCode).To(Equal(201))
					Expect(k).NotTo(BeNil())
					Expect(k.Id).To(Equal(AccessKeyId("anAccessKey")))
					Expect(k.SecretAccessKey).To(Equal(SecretAccessKey("aSecretAccessKey")))
					Expect(k.Email).To(Equal("user@example.com"))
					eT, _ := time.Parse(time.RFC3339, "2100-01-01T00:00:00Z")
					Expect(k.ExpiryDate).To(Equal(eT))
				})
			})
			Context("The ELS Returns another error", func() {
				JustBeforeEach(func() {
					server, sut = simServer(401, "")

					k, statusCode, err = sut.CreateAccessKey(ctx, email, password, false, expDays)
				})
				It("does not create an access key", func() {
					Expect(err).To(Equal(ErrUnexpectedStatusCode))
					Expect(statusCode).To(Equal(401))
					Expect(k).To(BeNil())
				})
			})
		})
	})
})
