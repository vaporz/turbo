.. _errorhandler:

ErrorHandler
============

By default, a HTTP code 500 error is returned if any error occurred.

You can customize this behavior via ErrorHandler:

.. code-block:: diff

 func InitComponents() {
 +	turbo.WithErrorHandler(errorHandler)
 }
 
 +func errorHandler(resp http.ResponseWriter, req *http.Request, err error) {
 +  	resp.Write([]byte("from errorHandler:" + err.Error()))
 +}

Restart and test(Modify "SayHello" to make it return an error)::

 $ curl -w "\n" "http://localhost:8081/hello?your_name=zx"
 from errorHandler:rpc error: code = Unknown desc = error!

