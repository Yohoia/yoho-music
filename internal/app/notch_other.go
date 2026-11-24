//go:build !darwin

package app

import "github.com/nsgo-dev/nsgo/core"

func sampleNotch(_ *core.Scope, done func(notchScreen)) { done(notchScreen{}) }
