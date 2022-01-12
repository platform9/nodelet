#!/opt/pf9/comms/nodejs/bin/node

/**
 * Copyright (c) 2016 Platform 9 Systems, Inc.
 */

'use strict';
const net = require('net');
const http = require('http');

function usage() {
    fatal('usage: portChecker.js timeout numClients ' +
         'port1 [port2 ... portN] uuid');
}

const numArgs = process.argv.length - 2;
if (numArgs < 4)
    usage();

const timeout = process.argv[2];
const numClients = Number(process.argv[3]);
const numPorts = process.argv.length - 5;
const uuid = process.argv[4 + numPorts];
let numListening = 0;
let statusPosted = false;
let totalConnectionsRemaining = numPorts * numClients;

log('uuid:', uuid);
let timer = setTimeout(() => postStatus(false, 'timeout'), timeout);
for (let i = 0; i < numPorts; ++i) {
    const port = process.argv[4 + i];
    log('registering port', port);
    listen(Number(port), onListening, numClients);
}

function onListening() {
    if (++numListening === numPorts) {
        postStatus(true, 'all ports listening');
    }
}


function listen(port, cb, remaining) {
    const srv = net.createServer(onConnect);
    srv.listen(port, cb);
    srv.once('error', err => fail(`server for port ${port} error: ${err}`));

    function onConnect(sock) {
        sock.once('error', err => fail(`socket for port ${port} error: ${err}`));
        sock.once('close', () => log('socket for port', port, 'closed'));
        log('incoming connection on port', port);
        if (remaining === 0)
            fail(`unexpected connection on port ${port}`);
        if (--remaining === 0)
            srv.close();
        if (--totalConnectionsRemaining === 0) {
            log('Done');
            process.exit(0); // don't wait for sockets to close
        }
    }
}

function fatal(msg) {
    logToStderr(msg);
    process.exit(1);
}

function fail(msg) {
    postStatus(false, msg);
}

function postStatus(ok, msg) {
    const body = { ok: ok, msg: msg };
    log('attempting to post status:', body);
    if (statusPosted)
        fatal(`status already posted, cannot post: ${msg}`);
    statusPosted = true;
    const opts = {
        method: 'POST',
        hostname: 'localhost',
        port: 9080,
        path: '/tackboard/',
        headers: {
            'content-type': 'application/json',
            'uuid': uuid
        }
    };
    const buf = new Buffer(JSON.stringify(body));
    const req = http.request(opts, onResponse);
    req.once('error', err => fatal(`http request error: ${err}`));
    req.write(buf);
    req.end();
    function onResponse(resp) {
        log('response status:', resp.statusCode);
        resp.once('error', err => fatal(`http response error: ${err}`));
        resp.on('data', (buf) => log('response data:', buf));
        resp.once('end', () => {
            log('response ended');
            if (!ok) // exit early in case of failure
                process.exit(0);
            if (numClients === 0) {
                log('There are zero clients. Exiting early.');
                process.exit(0);
            }
        });
    }
}

function log() {
    _log(arguments, 'log');
}

function logToStderr() {
    _log(arguments, 'error');
}

// Poor man's logging function with no dependencies. Inserts date.
function _log(args, fnName) {
    args = Array.from(args);
    const prefix = (new Date()).toString() + ' -';
    args.unshift(prefix);
    console[fnName].apply(console, args);
}

