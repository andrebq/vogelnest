// package trail provides a simple but effective way to write
// entries to a commit log.
//
// Each trail file is split into segments and each segment contain
// a sequence of entries.
//
// An entry is simply a header followed by an opaque sequence of
// bytes and a footer.
//
// Entries should be small as it might happen that a given node
// process then into memory. Segments should also be split into
// smaller chunks (couple of hundreds of MB's) as they might be
// sent back-and-forth over a network connection.
//
// Log files can be as large as the user wants, for smaller ones,
// the trail can be configured to automatically discard old segments.
//
// Note that currently it is not possible to split a segment into
// two segments
package trail
