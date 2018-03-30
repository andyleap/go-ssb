/*
This file is part of go-muxrpc.

go-muxrpc is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

go-muxrpc is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with go-muxrpc.  If not, see <http://www.gnu.org/licenses/>.
*/

// a simple RPC server for client tests
var MRPC = require('muxrpc')
var pull = require('pull-stream')
var toPull = require('stream-to-pull-stream')

var api = {
  hello: 'async',
  stuff: 'source'
}

var server = MRPC(null, api)({
  hello: function (name, name2, cb) {
    console.error('hello:ok')
    cb(null, 'hello, ' + name + ' and ' + name2 + '!')
  },
  stuff: function () {
    console.error('stuff called')
    return pull.values([{"a":1}, {"a":2}, {"a":3}, {"a":4}])
  }
})

var a = server.createStream()
pull(a, toPull.sink(process.stdout))
pull(toPull.source(process.stdin), a)
