.. _errorhandler:

ErrorHandler
============

默认配置下，有error发生时，会返回 HTTP 500 错误.

你可以通过 ErrorHandler 来定制行为:

.. code-block:: diff

 func InitComponents() {
 +	turbo.WithErrorHandler(errorHandler)
 }
 
 +func errorHandler(resp http.ResponseWriter, req *http.Request, err error) {
 +  	resp.Write([]byte("from errorHandler:" + err.Error()))
 +}

重启并测试(修改 "SayHello" 方法的代码，让它返回了一个error)::

 $ curl -w "\n" "http://localhost:8081/hello?your_name=zx"
 from errorHandler:rpc error: code = Unknown desc = error!

