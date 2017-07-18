.. _preprocessor:

Preprocessor
============

所有拦截器的Before()方法执行之后，向后端service发送请求之前，会执行Preprocessor。

Preprocessor可以用来执行一些某个API特有的逻辑，比如，参数校验，设置默认值，做类型转换，记录log等等。

下面的例子中，对参数'num'的值做了一些校验:

.. code-block:: diff

 func RegisterComponents(s *turbo.GrpcServer) {
 +	 s.RegisterComponent("preEatApple", preEatApple)
 }
 
 +var preEatApple turbo.Preprocessor = func (resp http.ResponseWriter, req *http.Request) error {
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

编辑 "yourservice/service.yaml":

.. code-block:: diff

 +preprocessor:
 +  - GET /eat_apple/{num:[0-9]+} preEatApple

重启服务并测试::

 $ curl -w "\n" "http://localhost:8081/eat_apple/5"
 message:"Good taste! Apple num=5"
 $ curl -w "\n" "http://localhost:8081/eat_apple/6"
 Too many apples!

