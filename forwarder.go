// SPDX-FileCopyrightText: 2019 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

// Forwarder define an interface to forward parsed HAProxy log to storage
// engine.
type Forwarder interface {
	Forwards(halogs []*HttpLog)
}
