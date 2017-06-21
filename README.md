# els-api-sdk-go
A collection of utilities written in Go for making API calls to the
Elastic Licensing Service (ELS).

## Introduction
All API calls must be [ELS-signed](https://docs.elasticlicensing.com/basics/api/els-signing).

In order to make API calls to the ELS from your own systems, you will need to
acquire an [Access Key](https://docs.elasticlicensing.com/basics/api/access-key)
for a user which has permission to make the API calls you wish to make.
Use the [ELS dashboard](https://dashboards.elasticlicensing.com) to manage users
and their access permissions.

### Access Key
Access Keys are obtained from the ELS for a specific user and contain
(among other things) the following information:

**Access Key ID** - a public string which forms part of the Authorization header
in ELS-signed requests.

**Secret Access Key** - a secret string which is used to generate the signature
part of the authorization header in ELS-signed requests.

### Obtaining an Access Key

Use the `APICaller.CreateAccessKey()` method. This method is provided as part of
the `APIUtils` interface (see `handler.go`).

### Signing and Sending a Request

Assuming you've created an `http.Request` object defining your request, use
`APICaller.Do()`.

Alternatively, if you wish to make a **GET** request to a URL, use `APICaller.Get()`

In both cases, a *profile* containing the Access Key should be passed.

For an example, see the implementation of the [els-cli](https://github.com/elasticlic/els-cli).


### Signing a request without Sending

Use `NewAPISigner(k *AccessKey)` to create a new APISigner which will sign
http.Requests with the given access key `k`.

Then use `APISigner.Sign(r *http.Request)` to sign request `r`.

**IMPORTANT**:

The request should be sent immediately after signing as the signature will
expire within a few minutes of the signature being generated.

## Troubleshooting

Common reasons for failure:

1. The time used in the signature is not accurate.
2. A request is signed but not sent till much later.
3. The user which the Access Key was generated for does not have permission to
make the API call.


# Versions

## Creating new versions
We're using [git flow](https://danielkummer.github.io/git-flow-cheatsheet/).


## Version History

### 1.0.0
*2017-06-21*

* Updated to use Sirupsen Logrus 1.0.0 (breaking change - Sirupsen -> sirupsen)
