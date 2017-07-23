.. _index: 

Turbo
=====

一个很"轻"的微服务框架。

主要功能
--------

* Turbo能生成一个反向代理服务器，把HTTP请求转换为 grpc 或者 Thrift 格式的请求。
* 支持 gRPC 和 :ref:`Thrift <thrift>`。
* 支持 :ref:`RESTFUL JSON API <json>`（"application/json"）。
* :ref:`拦截器（Interceptor） <interceptor>`。
* :ref:`预处理器（PreProcessor） <preprocessor>` 和 :ref:`后处理器（PostProcessor） <postprocessor>`: 可定制的URL-RPC映射过程。
* :ref:`劫持器（Hijacker） <hijacker>`: 接管整个request处理过程，你想干什么都行！
* :ref:`Message字段转换器（MessageFieldConvertor） <convertor>`: 告诉 Turbo 怎么给struct里的参数赋值。
* 不需要重启服务，在运行时修改和重新载入 :ref:`配置文件 <config>` ！
