package node_compat

import "code.dopame.me/veonik/squircy3/vm"

// EventEmitter is a polyfill for the node events module.
// See https://gist.github.com/mudge/5830382
// Modified to add listenerCount instance method.
var EventEmitter = &vm.Module{
	Name: "events",
	Main: "index",
	Path: "events",
	Body: `/* Polyfill EventEmitter. */
var EventEmitter = function () {
    this.events = {};
};

EventEmitter.EventEmitter = EventEmitter;

EventEmitter.prototype.on = function (event, listener) {
    if (typeof this.events[event] !== 'object') {
        this.events[event] = [];
    }

    this.events[event].push(listener);
};

EventEmitter.prototype.removeListener = function (event, listener) {
    var idx;

    if (typeof this.events[event] === 'object') {
        idx = this.events[event].indexOf(listener);

        if (idx > -1) {
            this.events[event].splice(idx, 1);
        }
    }
};

EventEmitter.prototype.emit = function (event) {
    var i, listeners, length, args = [].slice.call(arguments, 1);

    if (typeof this.events[event] === 'object') {
        listeners = this.events[event].slice();
        length = listeners.length;

        for (i = 0; i < length; i++) {
            listeners[i].apply(this, args);
        }
    }
};

EventEmitter.prototype.once = function (event, listener) {
    this.on(event, function g () {
        this.removeListener(event, g);
        listener.apply(this, arguments);
    });
};

EventEmitter.prototype.listenerCount = function (event) {
	if(!this.events[event]) {
		return 0;
	}
	return this.events[event].length;
};

module.exports = EventEmitter;`,
}
