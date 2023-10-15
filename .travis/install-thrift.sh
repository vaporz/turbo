#!/bin/sh -ex

PREFIX="$HOME/.thrift"
mkdir -p "$PREFIX"
if [ -x "$PREFIX/bin/thrift" ]; then
	"$PREFIX/bin/thrift" --version
	exit 0
fi

BUILD="$HOME/.thrift-build"
mkdir -p "$BUILD"
cd "$BUILD"

rm -rf thrift-0.19.0
ls

if [ ! -d thrift-0.19.0 ]; then
	sudo apt-get update -qq
	sudo apt-get install libboost-dev libboost-test-dev libboost-program-options-dev libevent-dev automake libtool flex bison pkg-config g++ libssl-dev
	wget http://archive.apache.org/dist/thrift/0.19.0/thrift-0.19.0.tar.gz
	tar -xzf thrift-0.19.0.tar.gz
	cd thrift-0.19.0
	./configure --enable-libs=no --enable-tests=no --enable-tutorial=no --without-c_glib --without-cpp --without-nodejs --without-haskell --without-java --without-python --without-ruby --without-perl --without-php --without-erlang
else
	cd thrift-0.19.0
fi

sudo make -j2
sudo make install

ls
