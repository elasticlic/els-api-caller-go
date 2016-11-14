package els

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Errors which may be expected to be returned from an APIHandler's methods.
var (
	ErrUnexpectedStatusCode = errors.New("Unexpected Status Code")
)

// APIUtils defines the methods which Api Handlers are expected to implement.
type APIUtils interface {
	CreateAccessKey(ctx context.Context, emailAddress string, password string, pwPrehashed bool, expiryDays uint) (*AccessKey, int, error)
}

// APIHandler implements APIUtils and provides convenience methods for
// interacting with the ELS API.
type APIHandler struct {
	// Scheme defines the http scheme to use - usually "https". In practise this
	// is only overriden during testing.
	Scheme string

	// Domain is the API domain, e.g. "api.elasticlicensing.com".
	Domain string

	// Version is the API version to use in requests. E.g. "1.0".
	Version string

	// Client is used to make all API calls.
	Client *http.Client
}

// NewAPIHandler returns an APIHandler configured to use the given http.Client.
// Pass nil for the http client, to force use of http.DefaultClient instead.
func NewAPIHandler(c *http.Client) *APIHandler {
	return &APIHandler{
		Scheme:  DefaultAPIScheme,
		Domain:  DefaultAPIDomain,
		Version: DefaultAPIVersion,
		Client:  c,
	}
}

// CreateAccessKey returns a new temporary AccessKey generated by the ELS, using
// the credentials passed. An AccessKey is used by a Signer to sign all ELS API
// calls. The credentials must match that of an existing user in the ELS.
// expiryDays determines after how many days the newly-generated access key
// should expire. If the context is cancelled or times out then ctx.Err() will
// be returned. If there is a response from the server but the http status code
// is not 201 (created), then an error will be returned and statusCode will
// indicate the statuscode received.
func (h *APIHandler) CreateAccessKey(ctx context.Context, emailAddress string, password string, pwPrehashed bool, expiryDays uint) (a *AccessKey, statusCode int, err error) {

	url := h.urlPrefix() + "/users/" + emailAddress + "/accessKeys?expires=1&numDaysTillExpiry=" + strconv.Itoa(int(expiryDays))

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, 0, err
	}

	if !pwPrehashed {
		// ELS requires clients to pre-hash all plaintext passwords.
		// Note that this hash is *NOT* what is stored in the ELS database.
		sh := sha256.Sum256([]byte(password))
		password = hex.EncodeToString(sh[:])
	}

	req.SetBasicAuth(emailAddress, password)

	log.WithFields(log.Fields{
		"Time":     time.Now(),
		"email":    emailAddress,
		"password": password,
		"auth":     req.Header["Authorization"],
		"req":      req,
	}).Debug("APIHandler: CreateAccessKey")

	rep, err := ctxhttp.Do(ctx, h.Client, req)
	if err != nil {
		return nil, 0, err
	}

	defer rep.Body.Close()

	if rep.StatusCode != http.StatusCreated {
		return nil, rep.StatusCode, ErrUnexpectedStatusCode
	}

	content, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return nil, 0, err
	}

	k := &AccessKey{}
	if err = json.Unmarshal(content, &k); err != nil {
		return nil, rep.StatusCode, err
	}

	return k, rep.StatusCode, nil
}

// urlPrefix returns the string to prepend to each relative API url.
func (h *APIHandler) urlPrefix() string {
	return h.Scheme + "://" + h.Domain + "/" + h.Version
}
