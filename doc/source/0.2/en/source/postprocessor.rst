.. _postprocessor:

Postprocessor
=============

By default, RPC response objects are format into a JSON string, and returned as API response.

Postprocessors handle responses from backend service. You can change default behavior by assigning a postprocessor.

Let's change the response of API "/eat_apple/{num:[0-9]+}":

Edit "yourservice/grpcapi/component/components.go":

.. code-block:: diff

 func RegisterComponents(s *turbo.GrpcServer) {
 +	 s.RegisterComponent("postEatApple", postEatApple)
 }

 +func postEatApple(resp http.ResponseWriter, req *http.Request, serviceResp interface{}) {
 +	sr := serviceResp.(*proto.EatAppleResponse)
 +	resp.Write([]byte("this is from postprocesser, message=" + sr.Message))
 +}

Edit "yourservice/service.yaml":

.. code-block:: diff

 +postprocessor:
 +  - GET /eat_apple/{num:[0-9]+} postEatApple

Restart HTTP server and test::

 $ curl -w "\n" "http://localhost:8081/eat_apple/5"
 this is from postprocesser, message=Good taste! Apple num=5

