# Sled 
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![GoDoc](https://godoc.org/github.com/Avalanche-io/sled?status.svg)](https://godoc.org/github.com/Avalanche-io/sled)
[![Stories in Ready](https://badge.waffle.io/Avalanche-io/sled.png?label=ready&title=Ready)](https://waffle.io/Avalanche-io/sled)
[![Build Status](https://travis-ci.org/Avalanche-io/sled.svg?branch=master)](https://travis-ci.org/Avalanche-io/sled)
[![Coverage Status](https://coveralls.io/repos/github/Avalanche-io/sled/badge.svg?branch=master)](https://coveralls.io/github/Avalanche-io/sled?branch=master)


Sled is a very high performance thread safe Key/Value store based on a _ctrie_ data structure with automatic persistence to disk via non-blocking snapshots.

Pre-defind accessors:

- Assets _the objects_
- Attributes _the metadata_

Versioned access via:

- Versions
- Branches 
- Tags

Also optional TTL, and Watcher interfaces to set _time to live_ for automatic key expiration, and notifications for key events.

## Notes

Key events include:

- Created
- Updated
- Saved
- Destroyed
- Expired

