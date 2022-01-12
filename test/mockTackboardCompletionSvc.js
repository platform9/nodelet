/**
 * Copyright (c) 2016 Platform 9 Systems, Inc.
 */

'use strict';
const http = require('http');
const events = require('events');
const emitter = new events.EventEmitter();
const streamBuffers = require('stream-buffers');

module.exports = function start(thePort, listenCb) {
    const port = thePort || 9080;
    const srv = http.createServer(onRequest);
    srv.listen(port, listenCb);
    return emitter;

    function onRequest(req, resp) {
        srv.close();
        req.once('error', err => fail(`request error: ${err}`));
        const sbuf = new streamBuffers.WritableStreamBuffer();
        req.pipe(sbuf);
        req.once('end', onEnd);
        sbuf.once('finish', onFinish);
        function onEnd() {
            resp.writeHead(200);
            resp.end();
        }
        function onFinish() {
            emitter.emit('response', JSON.parse(sbuf.getContents().toString()));
        }
    }
}


