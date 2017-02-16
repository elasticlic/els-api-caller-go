package els

import (
	"net/http"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/elasticlic/go-utils/datetime"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// APICaller identifies the methods that are used to access the ELS and other
// APIs.
type APICaller interface {
	// embedding APIUtils confers the ability to request an access key
	// for a user.
	APIUtils

	// Do executes the request, optionally ELS-signing it (if a signer is passed
	// and optionally completing the URL if the request is an ELS API request
	// (i.e. isELSAPI is true). Pass nil as ctx to use a default context or pass
	// your own if you want explicit control over the timeout period. Set
	// isELSAPI to false and pass nil as s if making an API call to a third
	// party.
	// If the request times out, error will be set to ctx.Err().
	Do(ctx context.Context, r *http.Request, s Signer, isELSAPI bool) (*http.Response, error)

	// Get executes an HTTP GET request with the given url. Pass nil as ctx to
	// use a default context or pass your own if you want explicit control over
	// the timeout period. If the request times out, error will be set to
	// ctx.Err().
	Get(ctx context.Context, url string, s Signer, isELSAPI bool) (*http.Response, error)

	// LastTimeout returns the time when an API call last failed to connect. If
	// there have been no timeouts, it will return the zero time (time.Time{})
	LastTimeout() time.Time
}

// EDAPICaller implements interface APICaller is used to make API calls to the
// ELS.
type EDAPICaller struct {
	sync.RWMutex

	// APIHandler is used to request Access Keys.
	APIHandler

	// tp is used to provide the time of 'now' used to sign requests. You can
	// simply pass time.Now() as an argument, or for testing, an object which
	// implements the TimeProvider interface to allow you to control the time
	// used.
	tp datetime.TimeProvider

	// lastTimeout stores the time when an attempt to make an API call last
	// timed-out.
	lastTimeout time.Time

	// requestTimeout governs how long to wait after making an API call before
	// giving up on the response.
	requestTimeout time.Duration
}

// NewEDAPICaller returns an EDAPICaller which will sign http.Requests and send them
// using the given client and signer.. Pass nil for c to use http.DefaultClient.
// Leave apiVersion blank to use the current version of the API. If you need to
// use multiple versions of the API, then create multiple EDAPICallers.
func NewEDAPICaller(c *http.Client, tp datetime.TimeProvider, timeout time.Duration, apiVersion string) (a *EDAPICaller) {
	a = &EDAPICaller{
		APIHandler:     *NewAPIHandler(c),
		tp:             tp,
		requestTimeout: timeout,
	}

	if apiVersion != "" && apiVersion != DefaultAPIVersion {
		a.APIHandler.Version = apiVersion
	}

	return a
}

// LastTimeout returns the last time a timeout was encountered by the
// EDAPICaller.
func (a *EDAPICaller) LastTimeout() time.Time {
	a.RLock()
	defer a.RUnlock()
	return a.lastTimeout
}

// Do completes the url of the request, signs the request and executes it. If
// the context has a deadline which expires, then context.DeadlineExceeded will
// be returned.
// Pass nil as ctx if you want a default context which times-out
// after the default ELS-signed API call timeout. Pass nil as s if you don't
// want the API call to be ELS-signed. Pass false as isELSAPI if the request
// is a call to a third-party API.
func (a *EDAPICaller) Do(ctx context.Context, r *http.Request, s Signer, isELSAPI bool) (*http.Response, error) {

	cancel := func() {}
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), a.requestTimeout)
	}
	defer cancel()

	if isELSAPI {
		u := r.URL
		u.Scheme = a.APIHandler.Scheme
		u.Host = a.APIHandler.Domain
		u.Path = "/" + a.APIHandler.Version + u.Path
	}

	// ELS-Sign the request
	if s != nil {
		if err := s.Sign(r, a.tp.Now()); err != nil {
			log.WithFields(log.Fields{"Time": time.Now(), "err": err}).Debug("ApiCaller: Failed to sign")
			return nil, err
		}
	}
	log.WithFields(log.Fields{"Time": time.Now(), "request": r}).Debug("ApiCaller: Do")
	resp, err := ctxhttp.Do(ctx, a.APIHandler.Client, r)

	if err != nil {
		t := a.tp.Now()
		a.Lock()
		a.lastTimeout = t
		a.Unlock()
		log.WithFields(log.Fields{"Time": t, "err": err, "response": resp}).Debug("ApiCaller: Timed out")
	}
	log.WithFields(log.Fields{"Time": time.Now(), "err": err, "response": resp}).Debug("ApiCaller: Response")

	return resp, err
}

// Get creates a signed GET request with a completed version of the url and
// executes it. Pass nil as ctx if you want a default context which times-out
// after the default ELS-signed API call timeout. Pass nill as s if you don't
// want the API call to be ELS-signed. Pass false as isELSAPI if the request
// is a call to a third-party API.
func (a *EDAPICaller) Get(ctx context.Context, url string, s Signer, isELSAPI bool) (*http.Response, error) {
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return a.Do(ctx, r, s, isELSAPI)
}
