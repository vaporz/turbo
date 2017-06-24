.. _create:

快速创建一个新的服务
====================

0, 开始之前
-----------

显然，你得先安装 :ref:`Golang <https://golang.org>` and :ref:`Protocol buffers(v3) <https://developers.google.com/protocol-buffers/>`
然后下载Turbo代码：

.. code-block:: shell

    go get github.com/vaporz/turbo

至于依赖管理，有两种方式：

(推荐) 安装 :ref:`glide <https://github.com/Masterminds/glide>` ::

    cd github.com/vaporz/turbo/turbo
    glide install

(不推荐) 手动安装这些包::

    go get google.golang.org/grpc
    go get git.apache.org/thrift.git/lib/go/thrift
    go get github.com/kylelemons/go-gypsy/yaml
    go get github.com/gorilla/mux
    go get github.com/spf13/cobra
    go get github.com/spf13/viper
    go get github.com/bitly/go-simplejson

1, 安装命令行工具
-----------------

::

    cd github.com/vaporz/turbo/protoc-gen-buildfields
    go install
    cd github.com/vaporz/turbo/turbo
    go install

2, 创建你的服务
----------------

::

 $ turbo create package/path/to/yourservice YourService -r grpc

文件夹 "$GOPATH/src/package/path/to/yourservice" 会被创建.

里面有一些生成的代码。

一个例子项目: https://github.com/vaporz/turbo-example/tree/master/yourservice

3, 运行
-------

好了！运行一个看看！

同时启动 gRPC server 和 HTTP server::

 cd $GOPATH/src/package/path/to/yourservice
 go run main.go

发送请求::

    $ curl -w "\n" "http://localhost:8081/hello?your_name=Alice"
    message:"Hello, Alice"

或者，你也可以分别启动 gRPC server 和 HTTP server::

    $ cd $GOPATH/src/package/path/to/yourservice
    # start grpc service
    $ go run grpcservice/yourservice.go
    # start http server
    $ go run grpcapi/yourserviceapi.go

