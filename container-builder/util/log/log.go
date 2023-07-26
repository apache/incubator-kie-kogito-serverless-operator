/*
 * Copyright 2022 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/api"
)

// Log --.
var Log Logger

func init() {
	Log = Logger{
		delegate: klog.NewKlogr().WithName(api.ComponentName),
	}
}

// Logger --.
type Logger struct {
	delegate klog.Logger
}

// Debugf --.
func (l Logger) Debugf(format string, args ...interface{}) {
	l.delegate.V(2).Info(fmt.Sprintf(format, args...))
}

// Infof --.
func (l Logger) Infof(format string, args ...interface{}) {
	l.delegate.Info(fmt.Sprintf(format, args...))
}

// Warnf --.
func (l Logger) Warnf(format string, args ...interface{}) {
	l.delegate.V(0).Info(fmt.Sprintf(format, args...))
}

// Errorf --.
func (l Logger) Errorf(err error, format string, args ...interface{}) {
	l.delegate.Error(err, fmt.Sprintf(format, args...))
}

// Debug --.
func (l Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.delegate.V(2).Info(msg, keysAndValues...)
}

// Info --.
func (l Logger) Info(msg string, keysAndValues ...interface{}) {
	l.delegate.Info(msg, keysAndValues...)
}

// Warn --.
func (l Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.delegate.V(0).Info(msg, keysAndValues...)
}

// Error --.
func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.delegate.Error(err, msg, keysAndValues...)
}

// WithName --.
func (l Logger) WithName(name string) Logger {
	return Logger{
		delegate: l.delegate.WithName(name),
	}
}

// WithName --.
func WithName(name string) Logger {
	return Log.WithName(name)
}

// Debugf --.
func Debugf(format string, args ...interface{}) {
	Log.Debugf(format, args...)
}

// Infof --.
func Infof(format string, args ...interface{}) {
	Log.Infof(format, args...)
}

// Warnf --.
func Warnf(format string, args ...interface{}) {
	Log.Warnf(format, args...)
}

// Errorf --.
func Errorf(err error, format string, args ...interface{}) {
	Log.Errorf(err, format, args...)
}

// Debug --.
func Debug(msg string, keysAndValues ...interface{}) {
	Log.Debug(msg, keysAndValues...)
}

// Info --.
func Info(msg string, keysAndValues ...interface{}) {
	Log.Info(msg, keysAndValues...)
}

// Warn --.
func Warn(msg string, keysAndValues ...interface{}) {
	Log.Warn(msg, keysAndValues...)
}

// Error --.
func Error(err error, msg string, keysAndValues ...interface{}) {
	Log.Error(err, msg, keysAndValues...)
}
