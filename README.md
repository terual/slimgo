slimgo squeezebox client
========================

slimgo only works on Linux, as it uses ALSA. If it doesn't compile, install the libasound2-dev package.

INSTALL:

1.  Get the latest go, see http://golang.org/doc/install.html and use the 'weekly' release.
2.  Set the $GOPATH environment variable, see http://tip.golang.org/cmd/go/#GOPATH_environment_variable
    If you use GOPATH=$HOME, sources will be put in `~/src`, packages in `~/pkg`, and binaries in `~/bin`
3.  Use `go get github.com/terual/slimgo` to build slimgo. The binary is located in `$GOPATH/bin/slimgo`

TODO:

-  Fix alignment issues, this causes sometimes static (watch out for your speakers!)

