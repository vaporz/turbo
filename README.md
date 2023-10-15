# Turbo  [![Build Status](https://travis-ci.org/vaporz/turbo.svg?branch=master)](https://travis-ci.org/vaporz/turbo) [![Coverage Status](https://coveralls.io/repos/github/vaporz/turbo/badge.svg?branch=master)](https://coveralls.io/github/vaporz/turbo?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/vaporz/turbo)](https://goreportcard.com/report/github.com/vaporz/turbo) [![codebeat badge](https://codebeat.co/badges/7a166e48-dae1-454c-b925-4fbcd3f1f461)](https://codebeat.co/projects/github-com-vaporz-turbo-master)

最新版本 | Latest Release: 0.5.0

文档地址 | Documentation: https://vaporz.github.io

-------------------------

I'm very happy and ready to help you if you're intersted in Turbo, and want to try it.<br>
Please create an issue if you have encountered any problems or have any new ideas. Thank you!<br>
如果你对Turbo感兴趣，并想试一试，我非常乐意帮助你。<br>
如遇到任何问题，或有新主意，请开issue，谢谢！<br>
![](https://github.com/vaporz/turbo/blob/image/Turbo.gif)(From movie "[Turbo](https://en.wikipedia.org/wiki/Turbo_(film))")

## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc/Thrift request.  
 **(In other words, now you have a grpc|thrift service? Turbo turns your grpc|thrift APIs into HTTP APIs!)**
 * Modify and reload [configuration](https://vaporz.github.io/master/en/config.html#config) file at runtime! Without restarting service.
 * Support gRPC and [Thrift](https://vaporz.github.io/master/en/thrift.html).
 * Support [RESTFUL JSON API](https://vaporz.github.io/master/en/json.html) ("application/json").
 * [Interceptor](https://vaporz.github.io/master/en/interceptor.html#interceptor).
 * [PreProcessor](https://vaporz.github.io/master/en/preprocessor.html#preprocessor) and [PostProcessor](https://vaporz.github.io/master/en/postprocessor.html#postprocessor): customizable URL-RPC mapping process.
 * [Hijacker](https://vaporz.github.io/master/en/hijacker.html#hijacker): Take over requests, do anything you want!
 * [Convertor](https://vaporz.github.io/master/en/convertor.html#convertor): Tell Turbo how to set a struct.
 * [Service Multiplexing](https://vaporz.github.io/master/en/multiplexing.html)
## Index
 * [Create a service on the fly](https://vaporz.github.io/master/en/create.html)
 * [Command line tools](https://vaporz.github.io/master/en/command.html)
 * [Rules and Conventions](https://vaporz.github.io/master/en/rules.html)
 * [How to add a new API](https://vaporz.github.io/master/en/add.html)
 * [Use a shared struct](https://vaporz.github.io/master/en/shared.html)
 * [Support RESTFUL JSON API](https://vaporz.github.io/master/en/json.html)
 * [Interceptor](https://vaporz.github.io/master/en/interceptor.html)
 * [PreProcessor](https://vaporz.github.io/master/en/preprocessor.html#preprocessor) and [PostProcessor](https://vaporz.github.io/master/en/postprocessor.html#postprocessor)
 * [Hijacker](https://vaporz.github.io/master/en/hijacker.html#hijacker)
 * [Convertor](https://vaporz.github.io/master/en/convertor.html#convertor)
 * [Error Handler](https://vaporz.github.io/master/en/errorhandler.html)
 * [Thrift support](https://vaporz.github.io/master/en/thrift.html)
 * [Configs in service.yaml](https://vaporz.github.io/master/en/config.html#config)
 * [Service Multiplexing](https://vaporz.github.io/master/en/multiplexing.html)
## Requirements
Golang version: >= 1.21.3 

Thrift version: 0.19.0  
