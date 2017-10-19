// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/interfaces/dbus"
	"github.com/snapcore/snapd/release"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/testutil"
)

type AvahiObserveInterfaceSuite struct {
	iface        interfaces.Interface
	plug         *interfaces.ConnectedPlug
	plugInfo     *snap.PlugInfo
	appSlot      *interfaces.ConnectedSlot
	appSlotInfo  *snap.SlotInfo
	coreSlot     *interfaces.ConnectedSlot
	coreSlotInfo *snap.SlotInfo
}

var _ = Suite(&AvahiObserveInterfaceSuite{
	iface: builtin.MustInterface("avahi-observe"),
})

const avahiObserveConsumerYaml = `name: consumer
apps:
 app:
  plugs: [avahi-observe]
`

const avahiObserveProducerYaml = `name: producer
apps:
 app:
  slots: [avahi-observe]
`

const avahiObserveCoreYaml = `name: core
slots:
  avahi-observe:
`

func (s *AvahiObserveInterfaceSuite) SetUpTest(c *C) {
	s.plug, s.plugInfo = MockConnectedPlug(c, avahiObserveConsumerYaml, nil, "avahi-observe")
	s.appSlot, s.appSlotInfo = MockConnectedSlot(c, avahiObserveProducerYaml, nil, "avahi-observe")
	s.coreSlot, s.coreSlotInfo = MockConnectedSlot(c, avahiObserveCoreYaml, nil, "avahi-observe")
}

func (s *AvahiObserveInterfaceSuite) TestName(c *C) {
	c.Assert(s.iface.Name(), Equals, "avahi-observe")
}

func (s *AvahiObserveInterfaceSuite) TestSanitizeSlot(c *C) {
	c.Assert(interfaces.SanitizeSlot(s.iface, s.coreSlotInfo), IsNil)
	c.Assert(interfaces.SanitizeSlot(s.iface, s.appSlotInfo), IsNil)
	// avahi-observe slot can now be used on snap other than core.
	slot := &snap.SlotInfo{
		Snap:      &snap.Info{SuggestedName: "some-snap"},
		Name:      "avahi-observe",
		Interface: "avahi-observe",
	}
	c.Assert(interfaces.SanitizeSlot(s.iface, slot), IsNil)
}

func (s *AvahiObserveInterfaceSuite) TestSanitizePlug(c *C) {
	c.Assert(interfaces.SanitizePlug(s.iface, s.plugInfo), IsNil)
}

func (s *AvahiObserveInterfaceSuite) TestAppArmorSpec(c *C) {
	// on a core system with avahi slot coming from a regular app snap.
	restore := release.MockOnClassic(false)
	defer restore()

	// connected plug to app slot
	spec := &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.appSlot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, "name=org.freedesktop.Avahi")
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `peer=(label="snap.producer.app"),`)
	// make sure observe does have observe but not control capabilities
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.AddressResolver`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.HostNameResolver`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.ServiceResolver`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.RecordBrowser`)
	// control capabilities
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `member=Set*`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `member=EntryGroupNew`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `interface=org.freedesktop.Avahi.EntryGroup`)

	// connected app slot to plug
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedSlot(s.iface, s.plug, s.appSlot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `interface=org.freedesktop.Avahi`)
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `peer=(label="snap.consumer.app"),`)
	// make sure observe does have observe but not control capabilities
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.AddressResolver`)
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.HostNameResolver`)
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.ServiceResolver`)
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.RecordBrowser`)
	// control capabilities
	c.Assert(spec.SnippetForTag("snap.producer.app"), Not(testutil.Contains), `interface=org.freedesktop.Avahi.EntryGroup`)

	// permanent app slot
	spec = &apparmor.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.appSlotInfo), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `dbus (bind)
    bus=system
    name="org.freedesktop.Avahi",`)

	// on a classic system with avahi slot coming from the core snap.
	restore = release.MockOnClassic(true)
	defer restore()

	// connected plug to core slot
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.coreSlot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, "name=org.freedesktop.Avahi")
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, "peer=(label=unconfined),")
	// make sure observe does have observe but not control capabilities
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.AddressResolver`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.HostNameResolver`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `interface=org.freedesktop.Avahi.ServiceResolver`)
	// control capabilities
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `member=Set*`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `member=EntryGroupNew`)
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Not(testutil.Contains), `interface=org.freedesktop.Avahi.EntryGroup`)

	// connected core slot to plug
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedSlot(s.iface, s.plug, s.coreSlot), IsNil)
	c.Assert(spec.SecurityTags(), HasLen, 0)

	// permanent core slot
	spec = &apparmor.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.coreSlotInfo), IsNil)
	c.Assert(spec.SecurityTags(), HasLen, 0)
}

func (s *AvahiObserveInterfaceSuite) TestDBusSpec(c *C) {
	// on a core system with avahi slot coming from a regular app snap.
	restore := release.MockOnClassic(false)
	defer restore()

	spec := &dbus.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.appSlotInfo), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `<allow own="org.freedesktop.Avahi"/>`)

	// on a classic system with avahi slot coming from the core snap.
	restore = release.MockOnClassic(true)
	defer restore()

	spec = &dbus.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.coreSlotInfo), IsNil)
	c.Assert(spec.SecurityTags(), HasLen, 0)
}

func (s *AvahiObserveInterfaceSuite) TestStaticInfo(c *C) {
	si := interfaces.StaticInfoOf(s.iface)
	c.Assert(si.ImplicitOnCore, Equals, false)
	c.Assert(si.ImplicitOnClassic, Equals, true)
	c.Assert(si.Summary, Equals, `allows discovery on a local network via the mDNS/DNS-SD protocol suite`)
	c.Assert(si.BaseDeclarationSlots, testutil.Contains, "avahi-observe")
}

func (s *AvahiObserveInterfaceSuite) TestAutoConnect(c *C) {
	// FIXME fix AutoConnect methods
	c.Assert(s.iface.AutoConnect(&interfaces.Plug{PlugInfo: s.plugInfo}, &interfaces.Slot{SlotInfo: s.coreSlotInfo}), Equals, true)
	c.Assert(s.iface.AutoConnect(&interfaces.Plug{PlugInfo: s.plugInfo}, &interfaces.Slot{SlotInfo: s.appSlotInfo}), Equals, true)
}

func (s *AvahiObserveInterfaceSuite) TestInterfaces(c *C) {
	c.Check(builtin.Interfaces(), testutil.DeepContains, s.iface)
}
