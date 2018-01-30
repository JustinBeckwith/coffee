# coffee

[![Greenkeeper badge](https://badges.greenkeeper.io/JustinBeckwith/coffee.svg)](https://greenkeeper.io/)

Coffee is an example of a go application written with the web framework [gin](https://github.com/gin-gonic/gin).  Bower is used for client side dependencies, gulp is used for the build.  To run locally, clone and run `gulp`.  Similar to a node.js application, the gulp file is set up to detect any changes in server side code, and automatically restart the server.  It's a nice workflow for go based web applications.

To get started:
- [Install Go](https://golang.org/doc/install)
- [Install Godep](https://github.com/tools/godep)
- [Install node.js](https://nodejs.org/)
- [Install gulp](https://github.com/gulpjs/gulp/blob/master/docs/getting-started.md)
- [Install bower](http://bower.io/#install-bower)

### gulp, bower, gin

This app uses gulp for the build, and bower for dependency management.  The gulpfile also watches for changes in any \*.go files and automatically restarts the server.  For more information, check out the [gulpfile](https://github.com/JustinBeckwith/coffee/blob/master/gulpfile.js).

To start the dev server, just run cd into the source directory and run `gulp`.

### godep

Go dependencies are tracked using [godep](https://github.com/tools/godep).  

- Install godep with `go get github.com/tools/godep`
- Run `godep save` after installing new dependencies with `go get`
- Use `godep go run` to run the application with the correct rewritten $GOPATH
