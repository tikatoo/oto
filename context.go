// Copyright 2021 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oto

import (
	"io"
	"sync"

	"github.com/hajimehoshi/oto/v2/internal/mux"
)

// Context is the main object in Oto. It interacts with the audio drivers.
//
// To play sound with Oto, first create a context. Then use the context to create
// an arbitrary number of players. Then use the players to play sound.
//
// Creating multiple contexts is NOT supported.
type Context struct {
	context *context
}

const (
	// FormatFloat32LE is the format of 32 bits floats little endian.
	FormatFloat32LE = 0

	// FormatUnsignedInt8 is the format of 8 bits integers.
	FormatUnsignedInt8 = 1

	//FormatSignedInt16LE is the format of 16 bits integers little endian.
	FormatSignedInt16LE = 2
)

// NewContext creates a new context, that creates and holds ready-to-use Player objects,
// and returns a context, a channel that is closed when the context is ready, and an error if it exists.
//
// Creating multiple contexts is NOT supported.
//
// The sampleRate argument specifies the number of samples that should be played during one second.
// Usual numbers are 44100 or 48000. One context has only one sample rate. You cannot play multiple audio
// sources with different sample rates at the same time.
//
// The channelCount argument specifies the number of channels. One channel is mono playback. Two
// channels are stereo playback. No other values are supported.
//
// The format argument specifies the format of sources.
// This value must be FormatFloat32LE, FormatUnsignedInt8, or FormatSignedInt16LE.
func NewContext(sampleRate int, channelCount int, format int) (*Context, chan struct{}, error) {
	ctx, ready, err := newContext(sampleRate, channelCount, mux.Format(format))
	if err != nil {
		return nil, nil, err
	}
	return &Context{context: ctx}, ready, nil
}

// NewPlayer creates a new, ready-to-use Player belonging to the Context.
// It is safe to create multiple players.
//
// The format of r is as follows:
//
//	[data]      = [sample 1] [sample 2] [sample 3] ...
//	[sample *]  = [channel 1] [channel 2] ...
//	[channel *] = [byte 1] [byte 2] ...
//
// Byte ordering is little endian.
//
// A player has some amount of an underlying buffer.
// Read data from r is queued to the player's underlying buffer.
// The underlying buffer is consumed by its playing.
// Then, r's position and the current playing position don't necessarily match.
// If you want to clear the underlying buffer for some reasons e.g., you want to seek the position of r,
// call the player's Reset function.
//
// You cannot share r by multiple players.
//
// The returned player implements Player, BufferSizeSetter, and io.Seeker.
// You can modify the buffer size of a player by the SetBufferSize function.
// A small buffer size is useful if you want to play a real-time PCM for example.
// Note that the audio quality might be affected if you modify the buffer size.
//
// If r does not implement io.Seeker, the returned player's Seek returns an error.
//
// NewPlayer is concurrent-safe.
//
// All the functions of a Player returned by NewPlayer are concurrent-safe.
func (c *Context) NewPlayer(r io.Reader) Player {
	return c.context.mux.NewPlayer(r)
}

// Suspend suspends the entire audio play.
//
// Suspend is concurrent-safe.
func (c *Context) Suspend() error {
	return c.context.Suspend()
}

// Resume resumes the entire audio play, which was suspended by Suspend.
//
// Resume is concurrent-safe.
func (c *Context) Resume() error {
	return c.context.Resume()
}

// Err returns the current error.
//
// Err is concurrent-safe.
func (c *Context) Err() error {
	return c.context.Err()
}

type atomicError struct {
	err error
	m   sync.Mutex
}

func (a *atomicError) TryStore(err error) {
	a.m.Lock()
	defer a.m.Unlock()
	if a.err == nil {
		a.err = err
	}
}

func (a *atomicError) Load() error {
	a.m.Lock()
	defer a.m.Unlock()
	return a.err
}
