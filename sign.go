package els

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	DefaultAPIScheme    = "https"
	DefaultAPIDomain    = "api.elasticlicensing.com"
	DefaultAPIVersion   = "1.0"
	RequiredContentType = "application/json;charset=utf-8"
)

var (
	ErrNoAccessKey       = errors.New("No Access Key")
	ErrNoRequest         = errors.New("No Request")
	ErrInvalidAccessKey  = errors.New("Invalid Access Key")
	ErrExpiredAccessKey  = errors.New("Expired Access Key")
	ErrRequestInvalidURL = errors.New("Invalid URL")
)

// Signer defines the methods that must be implemented by a class that
// implements ELS API request signing.
type Signer interface {
	Sign(r *http.Request, now time.Time) error
}

// APISigner implements the Signer interface and is used to modify an
// http.Request to be 'ELS-signed' by an Access Key (which is bound to an ELS
// user). ELS API calls must be ELS-signed or they will be immediately
// rejected. Note that even once ELS-Signed, a request may return an
// unauthorised response if the user whose AccessKey was used to sign the
// request is not authorised to make the request.
type APISigner struct {
	accessKey *AccessKey
}

func NewAPISigner(k *AccessKey) (a *APISigner, err error) {
	if k == nil {
		return nil, ErrNoAccessKey
	}
	a = &APISigner{
		accessKey: k,
	}

	return a, nil
}

// Sign signs the given request using the given access key. It is assumed that
// the request being signed will be sent immediately.
func (s *APISigner) Sign(r *http.Request, now time.Time) error {

	if r == nil {
		return ErrNoRequest
	}

	// We expect path to begin with the version of the API, which currently
	// must be 1.0
	if !strings.HasPrefix(r.URL.Path, "/1.0/") {
		return ErrRequestInvalidURL
	}

	k := s.accessKey

	if !k.ValidUntil(now, time.Minute) {
		return ErrExpiredAccessKey
	}

	utcStr := now.UTC().Format(time.RFC3339)

	ss := []string{r.Method, "\n"}

	if r.Body != nil {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		d := md5.Sum(b)
		md5s := hex.EncodeToString(d[:])
		ss = append(ss, md5s, "\n")
		ss = append(ss, RequiredContentType, "\n")
		// As we've read the body, we have to put things back so that it can
		// be read again when the request is eventually sent over the wire...
		// See https://medium.com/@xoen/golang-read-from-an-io-readwriter-without-loosing-its-content-2c6911805361#.pd96yml71
		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	} else {
		ss = append(ss, "\n\n")
	}

	ss = append(ss, utcStr, "\n")

	ss = append(ss, r.URL.Path)

	fingerprint := strings.Join(ss, "")

	h := hmac.New(sha256.New, []byte(k.SecretAccessKey))
	h.Write([]byte(fingerprint))

	hStr := base64.StdEncoding.EncodeToString(h.Sum(nil))

	auth := strings.Join([]string{"ELS ", string(k.Id), ":", hStr}, "")

	r.Header.Add("Authorization", auth)
	r.Header.Add("X-Els-Date", utcStr)
	r.Header.Add("Content-Type", RequiredContentType)

	log.WithFields(log.Fields{"Time": time.Now(), "fp": fingerprint, "auth": auth, "utcStr": utcStr}).Debug("Signer: sign")

	return nil
}
