#!/bin/bash

if [ x"$#" != x"1" ]; then
	echo "$0 subcmd"
	exit 1
fi
subcmd=$1; shift

repos=(
	github.com/expressjs/express		# JavaScript
	github.com/openclaw/openclaw		# TypeScript
	github.com/mermaid-js/mermaid		# JavaScript + TypeScript
	github.com/fastapi/fastapi		# Python
	github.com/jekyll/jekyll		# Ruby
	github.com/jaspervdj/hakyll		# Haskell
	github.com/janestreet/magic-trace	# OCaml
	github.com/mochi/mochiweb		# Erlang
	github.com/containers/container-libs	# Go
	github.com/youki-dev/youki		# Rust
	github.com/containers/libkrun		# Rust
	github.com/containers/crun		# C
	github.com/ceph/ceph			# C++
	github.com/grpc/grpc			# C++
)

ctags_cmd="/usr/local/u-ctags/bin/ctags -R --extras=+g --fields=+e"

topdir=${HOME}/devel/src
case ${subcmd} in
tags|maketags)
	for repo in "${repos[@]}"; do
		echo "=> ${repo}"
		(cd ${topdir}/${repo} && ${ctags_cmd})
		(cd ${topdir}/${repo} && ${ctags_cmd} --output-format=json > tags.json)
		echo
	done
	;;
jq)
	for repo in "${repos[@]}"; do
		echo "=> ${repo}"
		(cd ${topdir}/${repo} && jq '. | select(.name | contains("anonymousFunction") | not) | select(.kind == "function" or .kind == "method" or .kind == "func")' tags.json)
		echo
	done
	;;
*)
	echo "Unknown subcmd: ${subcmd}"
	exit 1
	;;
esac
