#!/opt/pf9/comms/nodejs/bin/node
'use strict';

var net = require('net');
if (process.argv.length != 4) {
    usage();
    process.exit(1);
}

var arg = [process.argv[2], process.argv[3]];
var options = [
    { host: arg[0].split(':')[0], port: arg[0].split(':')[1] },
    { path: arg[1] }
];

var sock = [null, null];

startConnection(0);
startConnection(1);

function startConnection(id) {
    var socket = net.connect(options[id], onConnect);
    socket.on('error', onError);
    socket.on('close', onClose);

    function onError(err) {
        console.error('Connection to', arg[id], 'experienced an error:', err);
    }

    function onClose(had_error) {
        var errStr = had_error? 'with error':'normally';
        console.info('Connection to', arg[id], 'closed', errStr, '.. exiting');
        process.exit(had_error? 2:0);
    }

    function onConnect() {
        console.log('Connected to', arg[id]);
        sock[id] = socket;
        var other = id ^ 1;
        if (sock[other]) {
            console.info('Joining sockets');
            socket.pipe(sock[other]).pipe(socket);
        }
    }
}

function usage() {
    console.log('unixsocket_forwarding_client.js host:port unix_domain_socket_path');
}
