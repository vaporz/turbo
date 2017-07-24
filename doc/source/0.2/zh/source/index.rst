.. _index: 

Turbo
=====

一个很"轻"的微服务框架。

主要功能
--------

* Turbo能生成一个反向代理服务器，把HTTP请求转换为 grpc 或者 Thrift 格式的请求。
* 支持 gRPC 和 `Thrift <thrift>`_ 。
* 支持 `RESTFUL JSON API <json>`_ （"application/json"）。
* `拦截器（Interceptor） <interceptor>`_ 。
* `预处理器（PreProcessor） <preprocessor>`_  和 `后处理器（PostProcessor） <postprocessor>`_ : 可定制的URL-RPC映射过程。
* `劫持器（Hijacker） <hijacker>`_ : 接管整个request处理过程，你想干什么都行！
* `Message字段转换器（MessageFieldConvertor） <convertor>`_ : 告诉 Turbo 怎么给struct里的参数赋值。
* 不需要重启服务，在运行时修改和重新载入 `配置文件 <config>`_  ！
