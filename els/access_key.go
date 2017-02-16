package els

import "time"

// AccessKeyID represents the public part of an ELS access Key.
type AccessKeyID string

// SecretAccessKey represents the private part of an ELS access Key.
type SecretAccessKey string

// AccessKey represents an access key that is used to sign ELS API Requests on
// behalf of a user identified by email address. An acesss key has a public
// 'accessKeyId' and a private 'secretAccessKey'.
type AccessKey struct {
	// Id is the public part of the access key which appears in the header of a
	// signed request. This field is mandatory.
	ID AccessKeyID `json:"accessKeyId"`

	// SecretAccessKey is the private part of the access key, known only by the
	// holder of the Key and the ELS, and whose value is used in the signing
	// process. This field is mandatory.
	SecretAccessKey SecretAccessKey `json:"secretAccessKey"`

	// ExpiryDate is an optional time which, if set to the non-zero time,  is
	// used to prevent use of the AccessKey to sign requests if it is known to
	// have expired.
	ExpiryDate time.Time `json:"expiryDt"`

	// Email is the email address of the user to whom this access key belongs.
	Email string `json:"emailAddress"`
}

// CanSign returns true if the AccessKey is able to sign an API Request.
func (a *AccessKey) CanSign() bool {
	return (a.ID) != "" && (a.SecretAccessKey) != ""
}

// ValidUntil returns true if the access key has not expired and will not do so
// within the given duration from now.
func (a *AccessKey) ValidUntil(now time.Time, in time.Duration) bool {

	return (a.ExpiryDate != time.Time{}) && (a.ExpiryDate.Sub(now) > in)
}
