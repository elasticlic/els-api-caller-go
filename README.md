# els-api-sdk-go
A collection of utilities written in Go for making API calls to the
Elastic Licensing Service (ELS).

## Introduction
All API calls must be 'ELS-signed' - I.e. they must have an **Authorization**
header which begins "ELS ". The content of the Authorization header is derived
from properties within an **Access Key** as well as other properties of the
request - e.g. the time sent, the content and the URL used.

In order to make API calls to the ELS from your own systems, you will need to
acquire an access key for a user which has permission to make the API calls you
wish to make. Use the ELS dashboard to manage users and their access
permissions.

### Access Key
Access Keys are obtained from the ELS for a specific user and contain
(among other things) the following information:

**Access Key ID** - a public string which forms part of the Authorization header
in ELS-signed requests.

**Secret Access Key** - a secret string which is used to generate the signature
part of the authorization header in ELS-signed requests.

### Obtaining an Access Key
Use the ApiHandler.CreateAccessKey() method.

### Signing a request
First, create an ApiSigner with an Access Key you've retrieved with
CreateAccessKey(). Then call ApiSigner.Sign(), passing your http.Request and
the current time. (The current time is always injected in order to help with
testing).


## Troubleshooting

An API call will be rejected by the ELS if the time you pass with now() is not
close to the real time (as reported by time.Time()).

An API call will also be rejected if the user to which the access key belongs
does not have permission to make the specific API call being attempted. You can
manage permissions for users using the ELS dashboards.
