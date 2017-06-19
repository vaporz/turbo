.. _preprocessor:

Preprocessor
============

What if I want to do something particularly for some API?

Preprocessor/Hijacker comes to help!

If both Preprocessors and hijackers are assigned to an URL, only the last hijacker assigned is active.

Preprocessor
------------

Preprocessors are executed just after all Before() functions from interceptors, and before sending requests to gRPC server.

Preprocessors can be used to do something particularly for an API. For example, parameter value validations, setting default values, parsing values, logging, etc.

Let's check the value of 'num' with a preprocessor:

.. code-block:: diff

 func InitComponents() {
 +	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", preEatApple)
 }
 
 +func preEatApple(resp http.ResponseWriter, req *http.Request) error {
 +	num,err := strconv.Atoi(req.Form["num"][0])
 +	if err!=nil {
 +		resp.Write([]byte("'num' is not numberic"))
 +		return errors.New("invalid num")
 +	}
 +	if num > 5 {
 +		resp.Write([]byte("Too many apples!"))
 +		return errors.New("Too many apples")
 +	}
 +	return nil
 +}

As usual, restart HTTP server, and test::

 $ curl -w "\n" "http://localhost:8081/eat_apple/5"
 message:"Good taste! Apple num=5"
 $ curl -w "\n" "http://localhost:8081/eat_apple/6"
 Too many apples!

