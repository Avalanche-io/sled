# Sled 

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

