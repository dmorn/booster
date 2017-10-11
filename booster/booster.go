// Package booster provides the core functionality of the whole booster concept.
// It is responsible for maintainig the `helper` devices group, collecting parallelizable
// requests and giving the appropriate work load to each device.
//
// Booster is not responsible for request/server checking, i.e. each request that it
// is going to process always comes togheter with the parallelisation procedure that
// booster should exploit, for example "take this request and multiplex it using"
// HTTP Bytes range headers.
package booster
