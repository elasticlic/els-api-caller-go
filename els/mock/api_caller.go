package mock

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/elasticlic/els-api-sdk-go/els"
	"golang.org/x/net/context"
)

// HTTPResponse creates a mock http.Response with the given statusCode and
// content body. Use with AddExpectedCall to simulate a response from the API.
func HTTPResponse(statusCode int, content string) *http.Response {
	r := &http.Response{StatusCode: statusCode}
	if content != "" {
		r.ContentLength = int64(len(content))
		r.Body = ioutil.NopCloser(bytes.NewReader([]byte(content)))
	}
	return r
}

// APICaller is a mock implementation of els.APICaller that allows us to record
// multiple requests and simulate a response for each in turn. It is useful in
// situations where there are expected to be a chain of API calls for which you
// want to check the arguments passed and simulate responses.
//
// To use, set up the test with one or more calls to AddExpectedCall() as many
// times as you expect an APICaller method to be invoked by the SUT. Specify
// the simulated response.
//
// Then after running the test, use GetCall to check the arguments sent to each
// invocation.
//
// If the expected method wasn't called by the SUT at any stage, or if the SUT
// makes more API calls than have been configured with AddExpectedCall() then
// APICaller will panic, and the test will fail.
type APICaller struct {
	sync.RWMutex

	// Calls stores the calls expected to be made - if more are encountered
	// than configured in this list, then a panic is thrown.
	Calls []*APICall

	// CallsMade records how many calls have been made by the SUT.
	CallsMade int

	// LastTo simulates the time the lastTimeout was encountered.
	LastTo time.Time
}

// NewAPICaller returns a new APICaller which implements interface
// core.APICaller.
func NewAPICaller() *APICaller {
	return &APICaller{
		Calls: []*APICall{},
	}
}

// AddExpectedCall adds an expectation of a call. The total number of expected
// calls is returned.
func (m *APICaller) AddExpectedCall(fn string, c APICall) int {
	m.Lock()
	defer m.Unlock()
	c.fn = fn
	m.Calls = append(m.Calls, &c)
	return len(m.Calls)
}

// GetCall returns a copy of the ith Call if it exists, or an error if it
// doesn't. exist. Use this after running the SUT to see what arguments each
// call was invoked with.
func (m *APICaller) GetCall(i int) *APICall {
	m.Lock()
	defer m.Unlock()
	l := len(m.Calls)
	if i >= l {
		panic(fmt.Sprintf("Call %d does not exist", i))
	}
	return m.Calls[i]
}

// AllCallsMade returns true if all the configured calls have been made.
func (m *APICaller) AllCallsMade() bool {
	m.Lock()
	defer m.Unlock()
	return m.CallsMade == len(m.Calls)
}

// NumCallsMade returns the number of calls that have been executed so far by
// the SUT.
func (m *APICaller) NumCallsMade() int {
	m.Lock()
	defer m.Unlock()

	return m.CallsMade
}

// CreateAccessKey implements interface core.APICaller
func (m *APICaller) CreateAccessKey(ctx context.Context, emailAddress string, password string, pwPrehashed bool, expiryDays uint) (*els.AccessKey, int, error) {
	a, r := m.initNextCall("CreateAccessKey")
	a.Context = ctx
	a.EmailAddress = emailAddress
	a.Password = password
	a.ExpiryDays = expiryDays
	a.PwPrehashed = pwPrehashed

	defer m.endCall()

	return r.AccessKey, r.StatusCode, r.Err
}

// Do implements interface core.APICaller
func (m *APICaller) Do(ctx context.Context, req *http.Request, s els.Signer, isELSAPI bool) (*http.Response, error) {
	a, r := m.initNextCall("Do")
	a.Context = ctx
	a.Req = req
	a.Signer = s
	a.IsELSAPI = isELSAPI

	defer m.endCall()

	return r.Rep, r.Err
}

// Get implements interface core.APICaller
func (m *APICaller) Get(ctx context.Context, URL string, s els.Signer, isELSAPI bool) (*http.Response, error) {
	a, r := m.initNextCall("Get")
	a.Context = ctx
	a.URL = URL
	a.Signer = s
	a.IsELSAPI = isELSAPI
	defer m.endCall()

	return r.Rep, r.Err
}

// LastTimeout implements interface core.APICaller
func (m *APICaller) LastTimeout() time.Time {
	return m.LastTo
}

// initNextCall is called at the start of processing of each call by the SUT
// to the APICaller.
func (m *APICaller) initNextCall(expectedFunc string) (*ACArgs, *ACRep) {
	m.Lock()

	if len(m.Calls) <= m.CallsMade {
		m.Unlock()
		panic(fmt.Sprintf("APICaller: Invoked by SUT too many times (num configured calls = %v)", len(m.Calls)))
	}

	c := m.Calls[m.CallsMade]
	if c.fn != expectedFunc {
		m.Unlock()
		panic(fmt.Sprintf("APICaller: Call #%v : Expected %s, was actually %s)", m.CallsMade+1, expectedFunc, c.fn))
	}
	c.CallMade = true
	time.Sleep(c.Delay)

	return &(c.ACArgs), &(c.ACRep)
}

// initNextCall is called at the end of processing of each call by the SUT
// to the APICaller.
func (m *APICaller) endCall() {
	defer m.Unlock()
	m.CallsMade++
}

// APICall represents a single expected call in an APICaller. It allows you to
// check the value of the arguments passed on the invocation this covers and
// you can stipulate the simulated response to be passed back. If you wish to
// simulate a timeout, then set error to context.DeadlineExceeded
type APICall struct {
	ACArgs
	ACRep

	// CallMade records whether the call was made
	CallMade bool

	// fn tells us which method is expected to be called
	fn string
}

// ACArgs represents the arguments an API call was invoked with.
type ACArgs struct {
	// Context is the context passed to use in the call.
	Context context.Context

	// EmailAddress is the email address presented to CreateAccessKey.
	EmailAddress string

	// Password is the email user password presented to CreateAccessKey.
	Password string

	// PwPrehashed is the pwPrehashed arg presented to CreateAccessKey.
	PwPrehashed bool

	// ExpiryDays is the expiryDays arg presented to CreateAccessKey.
	ExpiryDays uint

	// Req is the request presented to Do.
	Req *http.Request

	// URL is the URL passed to Get.
	URL string

	// Signer is the signer presented to Do or Get.
	Signer els.Signer

	// IsELSAPI stores the flag used to determine if calling the ELS API or a
	// third party API.
	IsELSAPI bool
}

// ACRep represents the simulated response to be returned when an API call
// is made.
type ACRep struct {
	// Delay is how long to wait until returning the simulated response when
	// a call is made.
	Delay time.Duration

	// Err is the error that should be returned from a call (E.g. to simulate
	// a call timing out, return context.DeadlineExceeded).
	Err error

	// Rep is the simulated http response that should be returned. Leave nil if
	// simulating a timeout error, or if simulating a response to
	// CreateAccessKey.
	Rep *http.Response

	// StatusCode is the statusCode arg returned from CreateAccessKey. Only set
	// this if simulating a response to a call to CreateAccessKey.
	StatusCode int

	// AccessKey is the AccessKey arg returns from CreateAccessKey.Only set
	// this if simulating a response to a call to CreateAccessKey.
	AccessKey *els.AccessKey
}
