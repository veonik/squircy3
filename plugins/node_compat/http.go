package main

import "code.dopame.me/veonik/squircy3/vm"

// Module Http is a polyfill for the node http module.
var Http = &vm.Module{
	Name: "http",
	Main: "index",
	Path: "http",
	Body: `
import EventEmitter from 'events';
import {Server as NetServer} from 'net';

export class Server extends NetServer {
    constructor(options, requestListener) {}

    setTimeout(timeout, callback = null) {}

    get maxHeadersCount() {}
    get timeout() {}
    get headersTimeout() {}
    get keepAliveTimeout() {}
}

export class OutgoingMessage {
    get upgrading() {}
    get chunkedEncoding() {}
    get shouldKeepAlive() {}
    get useChunkedEncodingByDefault() {}
    get sendDate() {}
    get finished() {}
    get headersSent() {}
    get connection() {}

    constructor() {}

    setTimeout(timeout, callback = null) {}
    setHeader(name, value) {}
    getHeader(name) {}
    getHeaders() {}
    getHeaderNames() {}
    hasHeader(name) {}
    removeHeader(name) {}
    addTrailers(headers) {}
    flushHeaders() {}
}

export class ServerResponse extends OutgoingMessage {
    get statusCode() {}
    get statusMessage() {}

    constructor(req) {}

    assignSocket(socket) {}
    detachSocket(socket) {}
    writeContinue(callback) {}
    writeHead(statusCode, reasonPhrase, headers = null) {}
}

export class ClientRequest extends OutgoingMessage {
    get connection() {}
    get socket() {}
    get aborted() {}

    constructor(uri, callback = null) {}

    get path() {}
    abort() {}
    onSocket(socket) {}
    setTimeout(timeout, callback = null) {}
    setNoDelay(noDelay) {}
    setSocketKeepAlive(enable, initialDelay = null) {}
}

class IncomingMessage {
    constructor(socket) {}

    get httpVersion() {}
    get httpVersionMajor() {}
    get httpVersionMinor() {}
    get connection() {}
    get headers() {}
    get rawHeaders() {}
    get trailers() {}
    get rawTrailers() {}
    setTimeout(timeout, callback = null) {}
    get method() {}
    get url() {}
    get statusCode() {}
    get statusMessage() {}
    get socket() {}

    destroy() {}
}

class Agent {
    get maxFreeSockets() {

    }
    get maxSockets() {}
    get sockets() {}
    get requests() {}

    constructor(options) {}

    destroy() {}
}

export const METHODS = [];
export const STATUS_CODES = {};

export function createServer(options, requestListener) {}

export function request(options, callback) {}
export function get(options, callback) {}

export let globalAgent;
export const maxHeaderSize;
`,
}
