.. _create:

Create a service on the fly
===========================

0, Before the start
-------------------

Obviously, you have to install `Golang <https://golang.org>`_ and `Protocol buffers(v3) <https://developers.google.com/protocol-buffers/>`_ first.

Then download turbo repo with:

.. code-block:: shell

    go get github.com/vaporz/turbo

For dependency management, we provide 2 ways:

(Recommended) Use `glide <https://github.com/Masterminds/glide>`_ ::

    cd github.com/vaporz/turbo/turbo
    glide install

(Not Recommended) Or install required packages manually::

    go get google.golang.org/grpc
    go get git.apache.org/thrift.git/lib/go/thrift
    go get github.com/kylelemons/go-gypsy/yaml
    go get github.com/gorilla/mux
    go get github.com/spf13/cobra
    go get github.com/spf13/viper
    go get github.com/bitly/go-simplejson

1, Install Turbo command line tools
-----------------------------------

::

    cd github.com/vaporz/turbo
    make

2, Create your service
----------------------

::

 $ turbo create package/path/to/yourservice YourService -r grpc

Directory "$GOPATH/src/package/path/to/yourservice" should appear.

There're also some generated files in this folder.

3, Run
------

That's it! Now let's Play!

Start both gRPC server and HTTP server::

 cd $GOPATH/src/package/path/to/yourservice
 go run main.go

Send a request::

    $ curl -w "\n" "http://localhost:8081/hello?your_name=Alice"
    message:"Hello, Alice"

Or you can start gRPC server and HTTP server separately::

    $ cd $GOPATH/src/package/path/to/yourservice
    # start grpc service
    $ go run grpcservice/yourservice.go
    # start http server
    $ go run grpcapi/yourserviceapi.go

