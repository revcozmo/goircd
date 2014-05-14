/*
goircd -- minimalistic simple Internet Relay Chat (IRC) server
Copyright (C) 2014 Sergey Matveev <stargrave@stargrave.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestRegistrationWorkflow(t *testing.T) {
	daemon := NewDaemon("foohost", "", nil, nil)
	events := make(chan ClientEvent)
	go daemon.Processor(events)
	conn := NewTestingConn()
	client := NewClient("foohost", conn)
	go client.Processor(events)

	conn.inbound <- "UNEXISTENT CMD" // should recieve nothing on this
	conn.inbound <- "NICK"

	if r := <-conn.outbound; r != ":foohost 431 :No nickname given\r\n" {
		t.Fatal("431 for NICK")
	}

	conn.inbound <- "NICK meinick\r\nUSER\r\n"
	if r := <-conn.outbound; r != ":foohost 461 meinick USER :Not enough parameters\r\n" {
		t.Fatal("461 for USER", r)
	}
	if (client.nickname != "meinick") || client.registered {
		t.Fatal("NICK saved")
	}

	conn.inbound <- "USER 1 2 3\r\n"
	if r := <-conn.outbound; r != ":foohost 461 meinick USER :Not enough parameters\r\n" {
		t.Fatal("461 again for USER")
	}

	daemon.SendLusers(client)
	if r := <-conn.outbound; !strings.Contains(r, "There are 0 users") {
		t.Fatal("LUSERS")
	}

	conn.inbound <- "USER 1 2 3 :4 5\r\n"
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 001") {
		t.Fatal("001 after registration")
	}
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 002") {
		t.Fatal("002 after registration")
	}
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 003") {
		t.Fatal("003 after registration")
	}
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 004") {
		t.Fatal("004 after registration")
	}
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 251") {
		t.Fatal("251 after registration")
	}
	if r := <-conn.outbound; !strings.Contains(r, ":foohost 422") {
		t.Fatal("422 after registration")
	}
	if (client.username != "1") || (client.realname != "4 5") || !client.registered {
		t.Fatal("client register")
	}

	conn.inbound <- "AWAY\r\n"
	conn.inbound <- "UNEXISTENT CMD\r\n"
	if r := <-conn.outbound; r != ":foohost 421 meinick UNEXISTENT :Unknown command\r\n" {
		t.Fatal("reply for unexistent command")
	}

	daemon.SendLusers(client)
	if r := <-conn.outbound; !strings.Contains(r, "There are 1 users") {
		t.Fatal("1 users logged in")
	}

	conn.inbound <- "QUIT\r\nUNEXISTENT CMD\r\n"
	<-conn.outbound
	if !conn.closed {
		t.Fatal("closed connection on QUIT")
	}
}

func TestMotd(t *testing.T) {
	fd, err := ioutil.TempFile("", "motd")
	if err != nil {
		t.Fatal("can not create temporary file")
	}
	defer os.Remove(fd.Name())
	fd.Write([]byte("catched\n"))

	conn := NewTestingConn()
	client := NewClient("foohost", conn)
	daemon := NewDaemon("foohost", fd.Name(), nil, nil)

	daemon.SendMotd(client)
	if r := <-conn.outbound; !strings.HasPrefix(r, ":foohost 375") {
		t.Fatal("MOTD start")
	}
	if r := <-conn.outbound; !strings.Contains(r, "372 * :- catched\r\n") {
		t.Fatal("MOTD contents")
	}
	if r := <-conn.outbound; !strings.HasPrefix(r, ":foohost 376") {
		t.Fatal("MOTD end", r)
	}
}
