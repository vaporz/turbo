.. _hijacker:

Hijacker
========

Hijacker跟preprocessor类型，区别在于，hijacker接管了整个request处理过程。

如果把一个hijacker配置给一个URL，它会在拦截器的最后一个Before()方法之后，第一个After()方法之前，接管这之间的整个处理过程。

你想干什么都行，这意味这，向service发送请求和错误处理这类事情，你也得自己来。

如果一个URL上同时设置了Hijacker和preprocessor，那么preprocessor会被忽略。

下面的例子中，URL "/eat_apple/{num:[0-9]+}" 被"接管"了，不管queryString里是什么值，参数"num"都被设置成了"999"。

.. code-block:: diff

 func RegisterComponents(s *turbo.GrpcServer) {
 +	 s.RegisterComponent("hijackEatApple", hijackEatApple)
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

编辑 "yourservice/service.yaml":

.. code-block:: diff

 +hijacker:
 +  - GET /eat_apple/{num:[0-9]+} hijackEatApple

重启服务并测试::

 $ curl -w "\n" "http://localhost:8081/eat_apple/6"
 message:"Good taste! Apple num=999"

