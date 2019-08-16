# squircy3

A proper IRC bot.


## Overview

squircy3 is a cross-platform application written in Go and should work just
about anywhere. Using a plugin architecture, the bot's capabilities and 
functionality are expandable to support pretty much anything.

Core plugins provide IRC client functionality, central configuration, and 
an embedded JavaScript runtime with support for ES6 and beyond.


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

