.. _postprocessor:

Postprocessor
=============

默认配置下，service返回的对象被转换成了JSON格式，然后直接作为API的返回值返回。

Postprocessor用来处理后端返回的对象（当然你也可以顺便干点别的事），你可以自由定制处理的方式。

下面例子中，将会修改 API "/eat_apple/{num:[0-9]+}" 的返回结果：

编辑 "yourservice/grpcapi/component/components.go":

.. code-block:: diff

 func RegisterComponents(s *turbo.GrpcServer) {
 +	 s.RegisterComponent("preEatApple", preEatApple)
 }
 
 +func postEatApple(resp http.ResponseWriter, req *http.Request, serviceResp interface{}) {
 +	sr := serviceResp.(*proto.EatAppleResponse)
 +	resp.Write([]byte("this is from postprocesser, message=" + sr.Message))
 +}

编辑 "yourservice/service.yaml":

.. code-block:: diff

 +postprocessor:
 +  - GET /eat_apple/{num:[0-9]+} postEatApple

重启服务并测试::

 $ curl -w "\n" "http://localhost:8081/eat_apple/5"
 this is from postprocesser, message=Good taste! Apple num=5

