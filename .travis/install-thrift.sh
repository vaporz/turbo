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
	./configure --enable-libs=no --enable-tests=no --enable-tutorial=no
else
	cd thrift-0.10.0
fi

sudo make -j2
sudo make install

ls
##!/bin/sh
#set -e
#set -x
#
#THRIFT_PREFIX="$HOME/.thrift"
#THRIFT_VER=0.10.0
#mkdir -p "$THRIFT_PREFIX"
#
#wget http://archive.apache.org/dist/thrift/${THRIFT_VER}/thrift-${THRIFT_VER}.tar.gz
#tar -xzvf thrift-${THRIFT_VER}.tar.gz
#cd thrift-${THRIFT_VER}
#./configure --prefix="$THRIFT_PREFIX" --enable-libs=no --enable-tests=no --enable-tutorial=no
#make -j2 && make install
#cd ${THRIFT_PREFIX}