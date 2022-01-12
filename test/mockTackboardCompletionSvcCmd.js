/**
 * Copyright (c) 2016 Platform 9 Systems, Inc.
 */

'use strict';
const mock = require('./mockTackboardCompletionSvc');
const emitter = mock(9080);
emitter.on('response', resp => console.log(resp));