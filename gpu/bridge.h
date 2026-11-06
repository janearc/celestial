// the C face of the bridge: what Go calls, in plain C types, so cgo
// never sees an Objective-C object. a session is an opaque handle;
// every call that can fail writes why into err and returns non-zero,
// and a call with no session says so instead of crashing.

#ifndef CELESTIAL_GPU_BRIDGE_H
#define CELESTIAL_GPU_BRIDGE_H

// a session is many kernels from one source, buffers that live on the
// device as long as it does, and batches of dispatches committed as one
// command, for work that runs many steps without coming back.

// celestial_gpu_open compiles the source once and makes a pipeline for
// each of count kernels, named in kernels. NULL on failure, with err
// saying why, and nodevice set to 1 when the reason is that there is no
// gpu at all.
void *celestial_gpu_open(const char *source, const char **kernels,
                         int count, int *nodevice, char *err, int errlen);

// celestial_gpu_buffer makes a shared buffer of length bytes, zeroed,
// and is its number, or -1 with err saying why.
int celestial_gpu_buffer(void *s, long length, char *err, int errlen);

// celestial_gpu_contents is where a buffer's bytes are, in memory the
// cpu and the gpu share; NULL with no session or for a number the
// session holds no buffer by. the bridge's own caller asks only for a
// buffer it has just made.
void *celestial_gpu_contents(void *s, int buffer);

// celestial_gpu_dispatch adds a dispatch of a kernel to the batch: its
// buffers bound from 0 in order, the params' bytes after them, over
// width by height threads. a buffer number the session does not hold,
// another session's say, is refused as "no such buffer".
int celestial_gpu_dispatch(void *s, int kernel, const int *buffers,
                           int count, const void *params, int paramlen,
                           int width, int height, char *err, int errlen);

// celestial_gpu_commit runs the batch, in order, waits for it, and
// begins another.
int celestial_gpu_commit(void *s, char *err, int errlen);

// celestial_gpu_seconds is the seconds the gpu has spent on every
// command the session has run, by the system's own timestamps on them:
// the time the work took on the gpu, and not the time the caller
// waited.
double celestial_gpu_seconds(void *s);

// celestial_gpu_close lets the session and its buffers go.
void celestial_gpu_close(void *s);

#endif
