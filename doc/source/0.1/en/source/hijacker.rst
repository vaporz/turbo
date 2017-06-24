.. _hijacker:

Hijacker
========

Hijackers are similar with preprocessors. The difference is, hijackers hijack the whole mapping process.

If a hijacker is assigned to an URL, it will take over the process between the last Before() and the first After() function.

You can do everything, which means you also have to call gRPC method yourself.

In this example, URL "/eat_apple/{num:[0-9]+}" is hijacked, no matter what the value is in query string, the value of parameter "num" is set to "999".

.. code-block:: diff

 func InitComponents() {
 +	turbo.SetHijacker("/eat_apple/{num:[0-9]+}", hijackEatApple)
 }
 
 +func hijackEatApple(resp http.ResponseWriter, req *http.Request) {
 +	client := turbo.GrpcService().(gen.YourServiceClient)
 +	r := new(gen.EatAppleRequest)
 +	r.Num = "999"
 +	res, err := client.EatApple(req.Context(), r)
 +	if err == nil {
 +		resp.Write([]byte(res.String() + "\n"))
 +	} else {
 +		resp.Write([]byte(err.Error() + "\n"))
 +	}
 +}

Restart and test::

 $ curl -w "\n" "http://localhost:8081/eat_apple/6"
 message:"Good taste! Apple num=999"

