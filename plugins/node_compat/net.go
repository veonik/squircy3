package node_compat

import "code.dopame.me/veonik/squircy3/vm"

// Net is a polyfill for the node net module.
var Net = &vm.Module{
	Name: "net",
	Main: "index",
	Path: "net",
	Body: `
import {Duplex} from 'stream';
import {Buffer} from 'buffer';
import {EventEmitter} from 'events';

const goAddrToNode = addr => {
	// todo: support ipv6 addresses, udp, ipc, etc
	let parts = addr.String().split(':');
	return {
		host: parts[0],
		port: parseInt(parts[1]),
		family: addr.Network(),
	};
};

export class Socket extends Duplex {
    constructor(options = {}) {
		super(options);
		this.options = options || {};
		this._connection = null;
		this._local = null;
		this._remote = null;
		this._connecting = false;
		this.on('ready', () => {
			this._connecting = false;
			this._local = goAddrToNode(this._connection.LocalAddr());
			this._remote = goAddrToNode(this._connection.RemoteAddr());
		});
    }

	_read(size = null) {
		if(!this._connection) {
			return;
		}
		let result = this._connection.Read(size);
		let wait = 1;
		let check = () => {
			if(result.Ready()) {
				let data = result.Value();
				if(data !== null && data.length) {
					this.push(data);
				} else {
					this.push(null);
				}
			} else {
				if(wait < 64) {
					wait *= 2;
				}
				setTimeout(check, wait);
			}
		};
		check();
	}

    _write(buffer, encoding, callback) {
		if(!this._connection) {
			callback(Error('not connected'));
			return;
		}
		let err = null;
		try {
			this._connection.Write(buffer);
		} catch(e) {
			err = e;
		} finally {
			callback(err);
		}
    }

    async connect(options, listener = null) {
		// todo: support ipc
		// todo: udp is defined in Node's dgram module
		if(listener !== null) {
			this.once('connect', listener);
		}
		let host = options.host || 'localhost';
		let port = options.port;
		if(!port) {
			throw new Error('ipc connections are unsupported');
		}
		console.log('dialing', host + ':' + port);
		this._connecting = true;
		this._connection = native.Dial('tcp', host + ':' + port);
		this.emit('connect');
		this.emit('ready');
    }

    setEncoding(encoding) {
		this.encoding = encoding;
    }

	_destroy(callback) {
		let err = null;
		try {
			this._connection.Close();
		} catch(e) {
			err = e;
			console.log('error destroying', err.toString());
		} finally {
			this._connection = null;
			if(callback) {
				callback(err);
			}
		}
	}

    // setTimeout(timeout, callback = null) {}
 	// setNoDelay(noDelay) {}
    // setKeepAlive(keepAlive) {}
    // address() {}
    // unref() {}
    // ref() {}
	// 
    // get bufferSize() {}
    // get bytesRead() {}
    // get bytesWritten() {}
    get connecting() {
		return this._connecting;
	}
    get localAddress() {
		if(!this._connection) {
			return null;
		}
		return this._local.host;
	}
    get localPort() {
		if(!this._connection) {
			return null;
		}
		return this._local.port;
	}
    get remoteAddress() {
		if(!this._connection) {
			return null;
		}
		return this._remote.host;
	}
    get remoteFamily() {
		if(!this._connection) {
			return null;
		}
		return this._remote.family;
	}
    get remotePort() {
		if(!this._connection) {
			return null;
		}
		return this._remote.port;
	}
}

export class Server extends EventEmitter {
    constructor(listener = null) {
		super();
		if(listener !== null) {
			this.on('connection', listener);
		}
		this._server = null;
    }

    listen(port, hostname, listener = null) {
		if(listener !== null) {
			this.on('connection', listener);
		}
		let addr = hostname + ':' + port;
		let accept = (conn) => {
			let socket = new Socket();
			socket._connection = conn;
			socket.on('end', () => {
				console.log('server ended');
				socket.destroy();
			});
			socket.emit('connect');
			this.emit('connection', socket);
			socket.emit('ready');
		};
		this._server = native.Listen(accept, 'tcp4', addr);
		this.emit('listening');
	}

	close(callback = null) {
		this._server.Close();
		this.emit('close');
		if(callback !== null) {
			callback();
		}
	}

    address() {
		return goAddrToNode(this._server.Addr());
	}

    getConnections(callback) {}

    ref() {}

    unref() {}

    get maxConnections() {}
    get connections() {}
    get listening() {}
}
// 
// export function createServer(options = null, connectionListener = null) {}
// 
// export function connect(options, connectionListener = null) {}
// 
// export function createConnection(options, connectionListener = null) {}
// 
// export function isIP(input) {}
// 
// export function isIPv4(input) {}
// 
// export function isIPv6(input) {}
`,
}
