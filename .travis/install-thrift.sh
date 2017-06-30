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

rm -rf thrift-0.10.0
ls

if [ ! -d thrift-0.10.0 ]; then
	wget http://archive.apache.org/dist/thrift/0.10.0/thrift-0.10.0.tar.gz
	tar -xzf thrift-0.10.0.tar.gz
	cd thrift-0.10.0
	./configure --enable-libs=no --enable-tests=no --enable-tutorial=no --without-haskell --without-java --without-python --without-ruby --without-perl --without-php --without-erlang
else
	cd thrift-0.10.0
fi

sudo make -j2
sudo make install

ls
