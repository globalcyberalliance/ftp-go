// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

// Notifier represents a notification operator interface.
type Notifier interface {
	BeforeCommand(ctx *Context, command string)
	BeforeLoginUser(ctx *Context, userName string)
	BeforePutFile(ctx *Context, dstPath string)
	BeforeDeleteFile(ctx *Context, dstPath string)
	BeforeChangeCurDir(ctx *Context, oldCurDir, newCurDir string)
	BeforeCreateDir(ctx *Context, dstPath string)
	BeforeDeleteDir(ctx *Context, dstPath string)
	BeforeDownloadFile(ctx *Context, dstPath string)
	AfterCommand(ctx *Context, command string, supported bool)
	AfterUserLogin(ctx *Context, userName, password string, passMatched bool, err error)
	AfterFilePut(ctx *Context, dstPath string, size int64, err error)
	AfterFileDeleted(ctx *Context, dstPath string, err error)
	AfterFileDownloaded(ctx *Context, dstPath string, size int64, err error)
	AfterCurDirChanged(ctx *Context, oldCurDir, newCurDir string, err error)
	AfterDirCreated(ctx *Context, dstPath string, err error)
	AfterDirDeleted(ctx *Context, dstPath string, err error)
}

type notifierList []Notifier

var _ Notifier = notifierList{}

func (notifiers notifierList) BeforeCommand(ctx *Context, command string) {
	for _, notifier := range notifiers {
		notifier.BeforeCommand(ctx, command)
	}
}

func (notifiers notifierList) BeforeLoginUser(ctx *Context, userName string) {
	for _, notifier := range notifiers {
		notifier.BeforeLoginUser(ctx, userName)
	}
}

func (notifiers notifierList) BeforePutFile(ctx *Context, dstPath string) {
	for _, notifier := range notifiers {
		notifier.BeforePutFile(ctx, dstPath)
	}
}

func (notifiers notifierList) BeforeDeleteFile(ctx *Context, dstPath string) {
	for _, notifier := range notifiers {
		notifier.BeforeDeleteFile(ctx, dstPath)
	}
}

func (notifiers notifierList) BeforeChangeCurDir(ctx *Context, oldCurDir, newCurDir string) {
	for _, notifier := range notifiers {
		notifier.BeforeChangeCurDir(ctx, oldCurDir, newCurDir)
	}
}

func (notifiers notifierList) BeforeCreateDir(ctx *Context, dstPath string) {
	for _, notifier := range notifiers {
		notifier.BeforeCreateDir(ctx, dstPath)
	}
}

func (notifiers notifierList) BeforeDeleteDir(ctx *Context, dstPath string) {
	for _, notifier := range notifiers {
		notifier.BeforeDeleteDir(ctx, dstPath)
	}
}

func (notifiers notifierList) BeforeDownloadFile(ctx *Context, dstPath string) {
	for _, notifier := range notifiers {
		notifier.BeforeDownloadFile(ctx, dstPath)
	}
}

func (notifiers notifierList) AfterCommand(ctx *Context, command string, supported bool) {
	for _, notifier := range notifiers {
		notifier.AfterCommand(ctx, command, supported)
	}
}

func (notifiers notifierList) AfterUserLogin(ctx *Context, userName, password string, passMatched bool, err error) {
	for _, notifier := range notifiers {
		notifier.AfterUserLogin(ctx, userName, password, passMatched, err)
	}
}

func (notifiers notifierList) AfterFilePut(ctx *Context, dstPath string, size int64, err error) {
	for _, notifier := range notifiers {
		notifier.AfterFilePut(ctx, dstPath, size, err)
	}
}

func (notifiers notifierList) AfterFileDeleted(ctx *Context, dstPath string, err error) {
	for _, notifier := range notifiers {
		notifier.AfterFileDeleted(ctx, dstPath, err)
	}
}

func (notifiers notifierList) AfterFileDownloaded(ctx *Context, dstPath string, size int64, err error) {
	for _, notifier := range notifiers {
		notifier.AfterFileDownloaded(ctx, dstPath, size, err)
	}
}

func (notifiers notifierList) AfterCurDirChanged(ctx *Context, oldCurDir, newCurDir string, err error) {
	for _, notifier := range notifiers {
		notifier.AfterCurDirChanged(ctx, oldCurDir, newCurDir, err)
	}
}

func (notifiers notifierList) AfterDirCreated(ctx *Context, dstPath string, err error) {
	for _, notifier := range notifiers {
		notifier.AfterDirCreated(ctx, dstPath, err)
	}
}

func (notifiers notifierList) AfterDirDeleted(ctx *Context, dstPath string, err error) {
	for _, notifier := range notifiers {
		notifier.AfterDirDeleted(ctx, dstPath, err)
	}
}

// NullNotifier implements Notifier.
type NullNotifier struct{}

var _ Notifier = &NullNotifier{}

// BeforeCommand implements Notifier.
func (NullNotifier) BeforeCommand(_ *Context, _ string) {}

// BeforeLoginUser implements Notifier.
func (NullNotifier) BeforeLoginUser(_ *Context, _ string) {}

// BeforePutFile implements Notifier.
func (NullNotifier) BeforePutFile(_ *Context, _ string) {}

// BeforeDeleteFile implements Notifier.
func (NullNotifier) BeforeDeleteFile(_ *Context, _ string) {}

// BeforeChangeCurDir implements Notifier.
func (NullNotifier) BeforeChangeCurDir(_ *Context, _ string, _ string) {}

// BeforeCreateDir implements Notifier.
func (NullNotifier) BeforeCreateDir(_ *Context, _ string) {}

// BeforeDeleteDir implements Notifier.
func (NullNotifier) BeforeDeleteDir(_ *Context, _ string) {}

// BeforeDownloadFile implements Notifier.
func (NullNotifier) BeforeDownloadFile(_ *Context, _ string) {
}

// AfterCommand implements Notifier.
func (NullNotifier) AfterCommand(_ *Context, _ string, _ bool) {}

// AfterUserLogin implements Notifier.
func (NullNotifier) AfterUserLogin(_ *Context, _ string, _ string, _ bool, _ error) {}

// AfterFilePut implements Notifier.
func (NullNotifier) AfterFilePut(_ *Context, _ string, _ int64, _ error) {}

// AfterFileDeleted implements Notifier.
func (NullNotifier) AfterFileDeleted(_ *Context, _ string, _ error) {}

// AfterFileDownloaded implements Notifier.
func (NullNotifier) AfterFileDownloaded(_ *Context, _ string, _ int64, _ error) {}

// AfterCurDirChanged implements Notifier.
func (NullNotifier) AfterCurDirChanged(_ *Context, _ string, _ string, _ error) {}

// AfterDirCreated implements Notifier.
func (NullNotifier) AfterDirCreated(_ *Context, _ string, _ error) {}

// AfterDirDeleted implements Notifier.
func (NullNotifier) AfterDirDeleted(_ *Context, _ string, _ error) {}
