'use strict';

const assert = require('assert');
const net = require('net');
const http = require('http');
const childProcess = require('child_process');
const portCheckerPath = '../root/opt/pf9/pf9-kube/diagnostics_utils/portChecker.js';
const log4js = require('log4js');
const log = log4js.getLogger('test');
const tackboardCompletionSvc = require('./mockTackboardCompletionSvc');
let tackboardEmitter = tackboardCompletionSvc(9080, test1);
const streamBuffers = require('stream-buffers');
let stdout;
let stderr = new streamBuffers.WritableStreamBuffer();
const responses = [];
const CONFLICTING_PORT = 8082;
const NUM_CLIENTS = 2;
let connectedPorts = 0;
tackboardEmitter.on('response', resp => responses.push(resp));

function test1() {
    // timeout test
    log.info('-- test1 --');
    const child = childProcess.spawn(process.execPath, [
        portCheckerPath,
        '1000',  // one second,
        '1',     // 1 client
        '8080',  // port
        'dummy-uuid' // uuid
    ]);
    child.once('exit', onTest1ChildExit);
    child.stdio[2].pipe(stderr);
}

function onTest1ChildExit(code) {
    assert.equal(code, 1);
    assert.equal(responses.length, 1);
    const resp = responses.pop();
    log.info('response:', resp);
    const stderrStr = stderr.getContents().toString();
    log.info('stderr:', stderrStr);
    assert.equal(resp.ok, true);
    assert.equal(resp.msg, 'all ports listening');
    assert(stderrStr.indexOf('timeout') >= 0);
    log.info('test1 ok');
    tackboardEmitter = tackboardCompletionSvc(9080, test2);
}

function test2() {
    // port conflict test
    log.info('-- test2 --');
    const conflictingSrv = net.createServer();
    conflictingSrv.listen(CONFLICTING_PORT, onListening);

    function onListening() {
        const child = childProcess.spawn(process.execPath, [
            portCheckerPath,
            '5000',  // five seconds,
            '1',     // 1 client
            CONFLICTING_PORT.toString(),  // port
            'dummy-uuid' // uuid
        ]);
        child.once('exit', onTest2ChildExit);
        stderr = new streamBuffers.WritableStreamBuffer();
        child.stderr.pipe(stderr);
    }

    function onTest2ChildExit(code) {
        assert.equal(code, 0);
        assert.equal(responses.length, 1);
        const resp = responses.pop();
        log.info('response:', resp);
        const stderrStr = stderr.getContents().toString();
        log.info('stderr:', stderrStr);
        assert.equal(resp.ok, false);
        assert(resp.msg.indexOf('EADDRINUSE') >= 0);
        conflictingSrv.close();
        log.info('test2 ok');
        tackboardEmitter = tackboardCompletionSvc(9080, test3);
    }
}

function test3() {
    // 2 port test
    log.info('-- test3 --');
    const child = childProcess.spawn(process.execPath, [
        portCheckerPath,
        '5000',  // five seconds,
        NUM_CLIENTS.toString(), // 2 clients
        '8080',  // port 1
        '8081',  // port 2
        'dummy-uuid' // uuid
    ]);
    child.once('exit', onTest3ChildExit);
    stdout = new streamBuffers.WritableStreamBuffer();
    child.stdout.pipe(stdout);
    tackboardEmitter.on('response', onResponse);
}

function onResponse() {
    assert.equal(responses.length, 1);
    const resp = responses.pop();
    log.info('response:', resp);
    assert.equal(resp.ok, true);
    assert.equal(resp.msg, 'all ports listening');
    let sock = net.connect(8080, function() {
        log.info('port 8080 connected 1st time');
        ++connectedPorts;
        sock.end();
    });
    sock.on('close', onFirstClose);
}

function onFirstClose() {
    let sock = net.connect(8081, function() {
        log.info('port 8081 connected 1st time');
        ++connectedPorts;
        sock.end();
    });
    sock.on('close', onSecondClose);
}

function onSecondClose() {
    log.info('2nd socket closed');
    let sock = net.connect(8080, function() {
        log.info('port 8080 connected 2nd time');
        ++connectedPorts;
        sock.end();
    });
    sock.on('close', onThirdClose);
}

function onThirdClose() {
    log.info('3rd socket closed');
    let sock = net.connect(8081, function() {
        log.info('port 8081 connected 2nd time');
        ++connectedPorts;
        sock.end();
    });
    sock.on('close', onFourthClose);
}

function onFourthClose() {
    log.info('4th socket closed');
    let sock = net.connect(8080, function() {
        log.error('port 8080 unexpectedly connected 3rd time');
        process.exit(1);
    });
    sock.on('error', err => log.info('Got expected connection error'));
}


function onTest3ChildExit(code) {
    assert.equal(code, 0);
    const stdoutStr = stdout.getContents().toString();
    log.info('stdout:', stdoutStr);
    assert.equal(2 * NUM_CLIENTS, connectedPorts);
    log.info('test3 ok');
}
