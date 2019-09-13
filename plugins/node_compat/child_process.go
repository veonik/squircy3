package node_compat

import "code.dopame.me/veonik/squircy3/vm"

var childProcess = &vm.Module{
	Name: "child_process",
	Main: "index",
	Body: `
import EventEmitter from 'events';

const runCommand = async (proc) => {
	proc.Run();
	await proc.Result();
	return proc;
}

export const spawn = (command, args, opts) => {
	if(typeof(exec) === 'undefined') {
		throw new Error("node_compat: cannot start child process: exec must be enabled");
	}
	let child = new class extends EventEmitter {
		constructor() {
			super();
			this.proc = exec.Command(command, ...(args || []));
			this.stdout = new EventEmitter();
			this.stderr = new EventEmitter();
			this.stdin = {
				write: (input) => {
					this.proc.Input(input);
				},
				end: () => {},
			};
		}
	};
	
	setTimeout(() => {
		runCommand(child.proc).catch((e) => {
			console.log('error running child_process', e);
		}).finally(res => {
			child.stdout.emit('data', child.proc.Output());
			child.stderr.emit('data', child.proc.Error());
			child.emit('close', child.proc.ExitCode());
		});
	}, 10);
	
	return child;
};
`,
}
