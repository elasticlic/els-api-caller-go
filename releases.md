# Version History

## 1.1.2
*2018-07-04*

* Fixed #3 - AccessKey.ValidUntil() does not work with non-expiring keys

## 1.1.1
*2017-08-16*

* Added 'omitempty' annotation for type AccessKey to reduce size of JSON
produced from this struct. Note that AccessKey contains a time.Time (a struct)
which cannot be omitted. The standard approach here is to turn the field into
a *time.Time pointer, which can be omitted with 'omitempty' but this has a
significant knock-on impact to users of AccessKey and so has not been changed.

## 1.1.0
*2017-07-31*

Issues Fixed:

* #2 | Signer: Incorrect signature for requests with non-nil empty bodies

## 1.0.0
*2017-06-21*

* Updated to use Sirupsen Logrus 1.0.0 (breaking change - Sirupsen -> sirupsen)
