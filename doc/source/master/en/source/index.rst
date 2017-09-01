.. _index: 

Turbo
=====

A lightweight microservice framework.

Features
--------

* Turbo generates a reverse-proxy server which translates a HTTP request into a grpc/Thrift request.
* Support gRPC and `Thrift <thrift>`_.
* Support `RESTFUL JSON API <json>`_ ("application/json").
* `Interceptor <interceptor>`_.
* `PreProcessor <preprocessor>`_ and `PostProcessor <postprocessor>`_: customizable URL-RPC mapping process.
* `Hijacker <hijacker>`_: Take over requests, do anything you want!
* `Convertor <convertor>`_: Tell Turbo how to set a struct field.
* Modify and reload `configuration <config>`_ file at runtime! Without restarting service.
