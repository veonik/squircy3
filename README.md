# squircy3

A proper IRC bot.


## Overview

squircy3 is a cross-platform application written in Go that works nearly
anywhere. 

squircy3 is also a framework based around a plugin architecture that can be
extended to support new and custom functionality.

Core plugins provide IRC client functionality, central configuration, and 
an embedded JavaScript runtime. Other built-in plugins provide NodeJS
compatibility including support for ES2017+ features through babel.js.


## Getting started

Clone this repository, then build using `make`.

```bash
git clone https://code.dopame.me/veonik/squircy3
cd squircy3
make all
```

The main `squircy` executable and all built plugins will be in `out/` after
a successful build.

Run `squircy` in interactive mode with `-interactive` and specify the `out/`
directory as the root.

> squircy creates the root directory if it does not yet exist. If the root
  does not contain config.toml or package.json, defaults will be created.

```bash
out/squircy -interactive -root out/config
```

This will automatically create `config.toml` and `package.json` within the 
`out/config/` directory. The default configuration enables all plugins available
and connects to libera.chat. As such, on the first run, expect to see some 
warnings.

> Expect to see some warnings on the first run. All plugins are enabled, by
  default, so NodeJS dependencies must be installed for everything to function.

Modify the default configuration in `out/config/config.toml` as necessary. Comment
out or remove the plugins listed under `extra_plugins` to disable any 
unwanted plugins.

To get the babel and node_compat plugins working, install Node dependencies 
using npm or yarn.

```bash
yarn --cwd out install
```

For more information on plugins, [see the section below](#Plugins)

### Docker

Run squircy3 in Docker using the `veonik/squircy3` image hosted on Docker Hub.

```bash
docker run -it veonik/squircy3:latest
```


## Configuration

squircy3 is configured using the TOML file `config.toml` below the bot's root 
directory.

> See [config.toml.dist](config.toml.dist) for the reference version of this file.

By default, the root directory is `~/.squircy`. If the root directory does not 
exist, it will be created and a default configuration will be populated.

squircy3 can also be configured using command line flags. Run `squircy -h` for
a full list of available options.


## Plugins

The `plugin` subpackage provides the interface for loading and managing plugins
within squircy3.

Plugins may be built-in to the binary application, or built as shared libraries
(.so files) that can be loaded at runtime.

### Core Plugins

Core plugins are built-in to the squircy application and are always loaded.

- `config` is a framework for pluggable, dynamic configuration management.
- `event` is an event dispatcher, allowing for decoupled communication between
  plugins and user scripts.
- `vm` is a javascript interpreter that supports ECMAScript 5.1 out of the box.
  - The `vm` plugin includes a mostly Node-compatible `require()` function.
  - This plugin also provides a concurrent-safe way to invoke some javascript 
    and retrieve the result whether it is sync or async.
- `irc` is an IRC client that utilizes the event dispatcher from the event 
  package to notify the application of messages, etc.

### Extra Plugins

Extra plugins are available to extend default functionality. These are built
as shared libraries and loaded at runtime using the Go plugin API.

- `babel` provides a transparent ES2017+ transpilation layer so that modern
  features can be used such as classes and async/await.
  - The `babel` plugin requires external NodeJS dependencies to operate.
    Use [package.json](package.json) as a starting point for installing these.
- `node_compat` adds an additional layer of compatibility with standard NodeJS
  APIs. 
  - This is extremely incomplete but currently supports: `event.EventEmitter`,
    NodeJS Streams (`stream`), `child_process.spawn()`, `crypto.Sha1` and 
    `crypto.createHash`, and basic forms of `net.Server` and `net.Socket`.
  - This plugin also loads the `regenerator-runtime` which allows Generators
    (ie. `function*` and `yield`) and async/await to function.
- `squircy2_compat` provides a compatibility layer with 
  [squircy2](https://squircy.com).
- `script` loads javascript files from a configured folder at app startup.
- `discord` provides integration with 
  [discordgo](https://github.com/bwmarrin/discordgo).

#### Linking extra plugins at compile-time

squircy3 supports building-in the extra plugins at compile-time so that they
are included in the main binary rather than as separate shared object files.

Pass `PLUGIN_TYPE=linked` to make to enable this functionality.

```bash
make all PLUGIN_TYPE=linked
```


## Related Projects

- **[squirssi](https://code.dopame.me/veonik/squirssi)** is an irssi clone using 
  squircy3 as the base framework. squirssi is actually just a squircy3 plugin and 
  thin wrapper around the squircy3/cli subpackage.
- [squircy2](https://github.com/veonik/squircy2) is the predecessor to squircy3.
