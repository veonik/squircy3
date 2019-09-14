# squircy3

A proper IRC bot.


## Overview

squircy3 is a cross-platform application written in Go that works nearly
anywhere. A plugin architecture enables the bot's capabilities and 
functionality to expand and support anything.

Core plugins provide IRC client functionality, central configuration, and 
an embedded JavaScript runtime. Other built-in plugins provide NodeJS
compatibility including support for ES6+ features through babel.


## Getting started

Clone this repository, then build using `make`.

```bash
git clone git@code.dopame.me:veonik/squircy3
cd squircy3
make all
```

The main `squircy` executable and all built plugins will be in `out/` after
a successful build.

Run `squircy`.

```bash
out/squircy
```

### Docker

Run squircy3 in Docker using the `veonik/squircy3` image hosted on Docker Hub.

```bash
docker run --it veonik/squircy3:latest
```

## Configuration

squircy3 is configured using the TOML file `config.toml` below the bot's root 
directory.

> See [config.toml.dist](config.toml.dist) for the reference version of this file.
