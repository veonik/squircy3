package main

import "code.dopame.me/veonik/squircy3/vm"

// Module Crypto is a polyfill providing some functionality from the node
// crypto module.
var Crypto = &vm.Module{
	Name: "crypto",
	Main: "index",
	Path: "crypto",
	Body: `
import {Buffer} from 'buffer';

class Sha1 {
	constructor() {
		this.data = Buffer.alloc(0);
	}

	update(data) {
		this.data.write(data);
		return this;
	}

	digest(kind) {
		switch(kind) {
			case 'hex':
				return sha1.Sum(this.data.toString());
			default:
				throw new Error('unsupported digest format: '+kind);
		}
	}
}

export const createHash = (kind) => {
	switch(kind) {
		case 'sha1':
			return new Sha1();
		default:
			throw new Error('unsupported hash algo: '+kind);
	}
};`,
}
