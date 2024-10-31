# ftp-go

[![GoDoc](https://godoc.org/github.com/globalcyberalliance/ftp-go?status.svg)](https://godoc.org/github.com/globalcyberalliance/ftp-go)

An FTP server implementation in Go.

_Forked from https://gitea.com/goftp/server ([Contributors](https://gitea.com/goftp/server/activity/contributors))._

## Installation

    go get github.com/globalcyberalliance/ftp-go

## Usage

To boot a FTP server you will need to provide a driver that speaks to your persistence layer - the required driver
contract is in [the documentation](http://pkg.go.dev/github.com/globalcyberalliance/ftp-go).

Look at the [file driver](https://goftp.io/server/driver/file) to see an example of how to build a backend.

And finally, connect to the server with any FTP client and the following
details:

    host: 127.0.0.1
    port: 2121
    username: admin
    password: 123456

This uses the file driver mentioned above to serve files.

## Warning

FTP is an incredibly insecure protocol. Avoid forcing users to authenticate with important credentials.

## License

This library is distributed under the terms of the MIT License. See the included file for more detail.

## Further Reading

There are a range of RFCs that together specify the FTP protocol. In chronological order, the more useful ones are:

* [http://tools.ietf.org/rfc/rfc959.txt](http://tools.ietf.org/rfc/rfc959.txt)
* [http://tools.ietf.org/rfc/rfc1123.txt](http://tools.ietf.org/rfc/rfc1123.txt)
* [http://tools.ietf.org/rfc/rfc2228.txt](http://tools.ietf.org/rfc/rfc2228.txt)
* [http://tools.ietf.org/rfc/rfc2389.txt](http://tools.ietf.org/rfc/rfc2389.txt)
* [http://tools.ietf.org/rfc/rfc2428.txt](http://tools.ietf.org/rfc/rfc2428.txt)
* [http://tools.ietf.org/rfc/rfc3659.txt](http://tools.ietf.org/rfc/rfc3659.txt)
* [http://tools.ietf.org/rfc/rfc4217.txt](http://tools.ietf.org/rfc/rfc4217.txt)

For an english summary that's somewhat more legible than the RFCs, and provides some commentary on what features are
actually useful or relevant 24 years after RFC959 was published:

* [http://cr.yp.to/ftp.html](http://cr.yp.to/ftp.html)

For a history lesson, check out Appendix III of RCF959. It lists the preceding(obsolete) RFC documents that relate to
file transfers, including the ye old RFC114 from 1971, "A File Transfer Protocol"

This library is heavily based on [em-ftpd](https://github.com/yob/em-ftpd), an FTPd framework with similar design goals
within the ruby and
EventMachine ecosystems.
