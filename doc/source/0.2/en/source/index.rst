.. _index: 

Turbo
=====

A lightweight microservice framework.

Golang version: >= 1.8.x

Features
--------

* Turbo generates a reverse-proxy server which translates a HTTP request into a grpc/Thrift request.
* Support gRPC and :ref:`Thrift <thrift>`.
* :ref:`Interceptor <interceptor>`.
* :ref:`PreProcessor <preprocessor>` and :ref:`PostProcessor <postprocessor>`: customizable URL-RPC mapping process.
* :ref:`Hijacker <hijacker>`: Take over requests, do anything you want!
* :ref:`MessageFieldConvertor <convertor>`: Tell Turbo how to set a struct field.
* Modify and reload :ref:`configuration <config>` file at runtime! Without restarting service.



