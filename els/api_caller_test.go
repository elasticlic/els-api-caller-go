package els

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/elasticlic/go-utils/datetime"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// DummySigner implements interface Signer
type DummySigner struct {
	LastRequest *http.Request
	LastSigned  time.Time
	ErrToReturn error
}

func (d *DummySigner) Sign(r *http.Request, now time.Time) error {
	d.LastRequest = r
	d.LastSigned = now
	r.Header.Add("Authorization", "some auth")
	r.Header.Add("X-Els-Date", now.UTC().Format(time.RFC3339))
	return d.ErrToReturn
}

var _ = Describe("HTTPResponse Suite", func() {

})

var _ = Describe("ApiCaller Suite", func() {

	var (
		sut              *EDAPICaller
		tp               = datetime.NewNowTimeProvider()
		httpClient       *http.Client
		apiVersion       string
		signer           Signer
		dummySigner      *DummySigner
		server           *httptest.Server
		req              *http.Request
		reqRec           *http.Request
		rep              *http.Response
		ctx              context.Context
		defaultCtx       = context.Background()
		dummyError       = errors.New("dummy error")
		route            string
		reqContent       = `{"some":"req"}`
		repContent       = `{"some":"data"}`
		err              error
		isELSAPI         bool
		now              time.Time
		timeout          = time.Second
		lastBodyReceived string

		// simServer creates a temporary http server, and modifies the SUT to
		// use the scheme and domain of the server, and configures the server
		// to respond with the given statuscode and repContent body after the given
		// duration.
		simServer = func(sut *EDAPICaller, statusCode int, body string, delay time.Duration) *httptest.Server {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqRec = r
				sbc, serr := ioutil.ReadAll(reqRec.Body)
				if serr == nil {
					lastBodyReceived = string(sbc)
				}

				time.Sleep(delay)
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
			h := &sut.APIHandler
			h.Client = c

			u, err := url.Parse(server.URL)
			if err != nil {
				log.Panic(err)
			}
			h.Scheme = u.Scheme
			h.Domain = u.Host
			return server
		}
	)

	log.SetOutput(ioutil.Discard)

	BeforeEach(func() {
		lastBodyReceived = ""
		apiVersion = ""
		httpClient = &http.Client{}
		dummySigner = &DummySigner{}
		signer = dummySigner
		ctx = nil
		isELSAPI = true
		route = "/path/to/route"
		now = time.Now()
		tp.SetNow(now)
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Describe("NewEDAPICaller", func() {
		JustBeforeEach(func() {
			sut = NewEDAPICaller(httpClient, tp, timeout, apiVersion)

		})
		It("Returns a correctly-defined EdApiCaller", func() {
			Expect(sut).NotTo(BeNil())
			Expect(sut.tp).To(Equal(tp))
			Expect(sut.requestTimeout).To(Equal(timeout))
			Expect(sut.APIHandler.Scheme).To(Equal("https"))
			Expect(sut.APIHandler.Domain).To(Equal("api.elasticlicensing.com"))
			Expect(sut.APIHandler.Version).To(Equal("1.0"))
			Expect(sut.APIHandler.Client).To(Equal(httpClient))
		})
		Context("The apiVersion is not left blank", func() {
			BeforeEach(func() {
				apiVersion = "1.2"
			})
			It("Uses the specified api version", func() {
				Expect(sut.APIHandler.Version).To(Equal("1.2"))
			})
		})
	})

	Describe("Do", func() {
		BeforeEach(func() {
			sut = NewEDAPICaller(httpClient, tp, timeout, apiVersion)
			server = simServer(sut, 200, repContent, 0)
		})

		AfterEach(func() {
			if rep != nil {
				rep.Body.Close()
			}
		})

		JustBeforeEach(func() {
			rep, err = sut.Do(ctx, req, signer, isELSAPI)
		})

		Context("A third party API is called", func() {
			BeforeEach(func() {
				isELSAPI = false
				route = "http://another.api.com/a/b"
				req, err = http.NewRequest("GET", route, nil)
				Expect(err).To(BeNil())
			})
			It("Does not change the URL", func() {
				Expect(err).To(BeNil())
				bc, cerr := ioutil.ReadAll(rep.Body)
				Expect(cerr).To(BeNil())
				Expect(reqRec.URL.Scheme).To(Equal("http"))
				Expect(reqRec.URL.Path).To(Equal("/a/b"))
				Expect(reqRec.URL.Host).To(Equal("another.api.com"))
				Expect(bc).Should(MatchJSON(repContent))
			})
		})

		Context("A request is attempted with no body", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", route, nil)
				Expect(err).To(BeNil())
			})
			It("Completes the URL and returns the expected response", func() {
				Expect(rep).NotTo(BeNil())
				Expect(rep.StatusCode).To(Equal(200))
				Expect(err).To(BeNil())
				// Check we get back what we asked for
				Expect(rep.Body).NotTo(BeNil())
				bc, cerr := ioutil.ReadAll(rep.Body)
				Expect(cerr).To(BeNil())
				Expect(reqRec.URL.Path).To(Equal("/" + DefaultAPIVersion + route))
				Expect(bc).Should(MatchJSON(repContent))
			})

			Context("The signer cannot sign the request", func() {
				BeforeEach(func() {
					dummySigner.ErrToReturn = dummyError
				})
				It("Returns the signer error", func() {
					Expect(err).To(Equal(dummyError))
				})
			})

			Context("The request times out", func() {
				BeforeEach(func() {
					// force an idiotically-short timeout
					sut.requestTimeout = time.Nanosecond
					// force the response to take longer than this - we'll have to
					// rebuild the server for this test:
					server.Close()
					server = simServer(sut, 200, repContent, time.Millisecond)
				})

				It("returns a timeout error", func() {
					Expect(err).To(Equal(context.DeadlineExceeded))
				})
			})

			Context("A context is presented", func() {
				// We want to check that if a context is presented, that this is
				// used rather than an internally-generated one with the default
				// timeout.
				BeforeEach(func() {
					ctx = defaultCtx
				})
				JustBeforeEach(func() {
					bc, cerr := ioutil.ReadAll(rep.Body)
					Expect(cerr).To(BeNil())
					Expect(reqRec.URL.Path).To(Equal("/" + DefaultAPIVersion + route))
					Expect(bc).Should(MatchJSON(repContent))
				})
			})
			Context("A context is not presented", func() {
				// We expect an internally-generated context with the timeout
				// defined in the EDConfig to be created and used.
				BeforeEach(func() {
				})

				Context("The request completes before timeout", func() {
					It("returns the response", func() {
						Expect(rep).NotTo(BeNil())
						bc, cerr := ioutil.ReadAll(rep.Body)
						Expect(cerr).To(BeNil())
						Expect(reqRec.URL.Path).To(Equal("/" + DefaultAPIVersion + route))
						Expect(bc).Should(MatchJSON(repContent))
					})
				})
			})
		})
		Context("A signed request is attempted with a body", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("POST", route, bytes.NewBuffer([]byte(reqContent)))
				Expect(err).To(BeNil())

			})
			It("Submits the data", func() {
				Expect(rep).NotTo(BeNil())
				Expect(rep.StatusCode).To(Equal(200))
				Expect(err).To(BeNil())
				// Check the server received our body content
				Expect(lastBodyReceived).Should(MatchJSON(reqContent))

				// Check we get back what we asked for
				Expect(rep.Body).NotTo(BeNil())
				bc, cerr := ioutil.ReadAll(rep.Body)
				Expect(cerr).To(BeNil())
				Expect(reqRec.URL.Path).To(Equal("/" + DefaultAPIVersion + route))
				Expect(bc).Should(MatchJSON(repContent))
			})
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			sut = NewEDAPICaller(httpClient, tp, timeout, apiVersion)
			server = simServer(sut, 200, repContent, 0)
		})

		JustBeforeEach(func() {
			rep, err = sut.Get(nil, route, signer, isELSAPI)
		})
		AfterEach(func() {
			if rep != nil {
				rep.Body.Close()
			}
		})
		It("Executes the request", func() {
			Expect(err).To(BeNil())
			bc, cerr := ioutil.ReadAll(rep.Body)
			Expect(cerr).To(BeNil())
			Expect(reqRec.URL.Path).To(Equal("/" + DefaultAPIVersion + route))
			Expect(bc).Should(MatchJSON(repContent))
		})
	})
})
